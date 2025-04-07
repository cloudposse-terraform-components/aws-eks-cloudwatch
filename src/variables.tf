variable "region" {
  description = "AWS Region"
  type        = string
}

variable "priority_class_enabled" {
  type        = bool
  description = "Whether to enable the priority class for the EKS addon"
  default     = false
}

# variable "chart" {
#   type        = string
#   description = "The Helm chart to install"
#   default     = "amazon-cloudwatch-observability"
# }
#
# variable "chart_repository" {
#   type        = string
#   description = "Repository URL where to locate the requested chart."
#   default     = "https://aws-observability.github.io/helm-charts"
# }

variable "chart_description" {
  type        = string
  description = "Set release description attribute (visible in the history)"
  default     = "Amazon CloudWatch Observability for EKS"
}

variable "kubernetes_namespace" {
  type        = string
  description = "Name of the Kubernetes Namespace this pod is deployed in to"
  default     = "amazon-cloudwatch"
}

variable "timeout" {
  type        = number
  description = "Time in seconds to wait for any individual kubernetes operation (like Jobs for hooks)"
  default     = 900 # 15 minutes
}

variable "cleanup_on_fail" {
  type        = bool
  description = "Allow deletion of new resources created in this upgrade when upgrade fails"
  default     = true
}

variable "atomic" {
  type        = bool
  description = "If set, installation process purges chart on fail. The wait flag will be set automatically if atomic is used"
  default     = true
}

variable "wait" {
  type        = bool
  description = "Will wait until all resources are in a ready state before marking the release as successful. It will wait for as long as `timeout`"
  default     = null
}

variable "chart_values" {
  type        = any
  description = "Additional values to yamlencode as `helm_release` values"
  default     = {}
}
