locals {
  enabled = module.this.enabled

  priority_class_enabled = local.enabled && var.priority_class_enabled
  priority_class_name    = module.local.id

  kubernetes_labels = {
    for k, v in merge(module.this.tags, { name = var.kubernetes_namespace }) : k => replace(v, "/", "_")
    if local.enabled
  }

  # If karpenter is enabled, you may need to add the karpenter iam role to the list of worker roles
  eks_worker_role_names = compact(module.eks.outputs.eks_node_group_role_names)
}

module "local" {
  source  = "cloudposse/label/null"
  version = "0.25.0"

  name    = length(module.this.name) > 0 ? module.this.name : "cloudwatch"
  context = module.this.context
}

module "cloudwatch" {
  source  = "cloudposse/helm-release/aws"
  version = "0.10.1"

  #chart       = var.chart
  #repository  = var.chart_repository
  # The upstream helm chart does not support priority class, which we need to schedule pods on all nodes.
  # Therefore, we need to fork the chart and add the priority class to the deployment.
  chart       = "${path.module}/charts/amazon-cloudwatch-observability"
  description = var.chart_description

  wait            = var.wait
  atomic          = var.atomic
  cleanup_on_fail = var.cleanup_on_fail
  timeout         = var.timeout

  create_namespace_with_kubernetes = true
  kubernetes_namespace             = var.kubernetes_namespace
  kubernetes_namespace_labels      = local.kubernetes_labels

  eks_cluster_oidc_issuer_url = replace(module.eks.outputs.eks_cluster_identity_oidc_issuer, "https://", "")

  iam_role_enabled = false

  # Chart values
  values = compact([
    yamlencode(merge(
      {
        fullnameOverride = module.local.name,
        clusterName      = one(module.eks.outputs[*].eks_cluster_id),
        region           = var.region,
      },
      local.priority_class_enabled ? { priorityClassName = local.priority_class_name } : {}
    )),

    # additional values
    try(length(var.chart_values), 0) == 0 ? null : yamlencode(var.chart_values)
  ])

  context = module.local.context

  depends_on = [kubernetes_priority_class.this]
}

resource "aws_iam_role_policy_attachment" "this" {
  for_each = local.enabled ? toset(local.eks_worker_role_names) : toset([])

  role       = each.value
  policy_arn = "arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy"
}

resource "kubernetes_priority_class" "this" {
  count = local.priority_class_enabled ? 1 : 0

  metadata {
    name = local.priority_class_name
  }

  value          = 1000000
  global_default = false
  description    = "Priority class for the ${module.local.id} EKS CloudWatch Helm chart"
}
