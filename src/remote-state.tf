variable "eks_component_name" {
  description = "The name of the EKS component"
  type        = string
  default     = "eks/cluster"
}

module "eks" {
  source  = "cloudposse/stack-config/yaml//modules/remote-state"
  version = "1.5.0"

  component = var.eks_component_name

  context = module.this.context
}
