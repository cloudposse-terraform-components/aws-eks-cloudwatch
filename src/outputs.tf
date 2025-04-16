locals {
  # These are the default values created by the chart
  cloudwatch_log_group_names = [
    "/aws/containerinsights/${one(module.eks.outputs[*].eks_cluster_id)}/application",
    "/aws/containerinsights/${one(module.eks.outputs[*].eks_cluster_id)}/dataplane",
    "/aws/containerinsights/${one(module.eks.outputs[*].eks_cluster_id)}/host",
    "/aws/containerinsights/${one(module.eks.outputs[*].eks_cluster_id)}/performance"
  ]
}

output "metadata" {
  value       = module.cloudwatch.metadata
  description = "Block status of the deployed release"
}

output "cloudwatch_log_group_names" {
  value       = local.cloudwatch_log_group_names
  description = "List of CloudWatch log group names created by the agent"
}

output "priority_class_name" {
  value       = local.priority_class_enabled ? local.priority_class_name : ""
  description = "Name of the priority class"
}
