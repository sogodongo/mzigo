variable "name_prefix" {
  description = "Prefix applied to all resource names in this module."
  type        = string
}

variable "vpc_cidr" {
  description = "CIDR block for the VPC."
  type        = string
}

variable "availability_zones" {
  description = "List of AZs to deploy subnets into."
  type        = list(string)
}
