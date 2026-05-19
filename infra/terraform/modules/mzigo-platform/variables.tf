variable "name_prefix" {
  description = "Prefix applied to all resource names."
  type        = string
}

variable "environment" {
  description = "Deployment environment."
  type        = string
}

variable "aws_region" {
  description = "AWS region."
  type        = string
}

variable "eks_cluster_name" {
  description = "Name of the EKS cluster. Used for IRSA trust policy construction."
  type        = string
}

variable "vpc_id" {
  description = "VPC ID. Received from the networking module."
  type        = string
}

variable "private_subnet_ids" {
  description = "Private subnet IDs for EKS workloads."
  type        = list(string)
}

variable "intra_subnet_ids" {
  description = "Intra subnet IDs for MSK brokers and RDS."
  type        = list(string)
}

variable "kafka_version" {
  description = "Apache Kafka version for MSK."
  type        = string
}

variable "broker_instance_type" {
  description = "MSK broker EC2 instance type."
  type        = string
}

variable "broker_count" {
  description = "Number of MSK broker nodes."
  type        = number
}

variable "broker_storage_gb" {
  description = "EBS storage per MSK broker in GB."
  type        = number
}

variable "rds_instance_class" {
  description = "RDS instance class."
  type        = string
}

variable "rds_allocated_storage_gb" {
  description = "Initial RDS storage in GB."
  type        = number
}

variable "rds_postgres_version" {
  description = "PostgreSQL version."
  type        = string
}

variable "lineage_retention_days" {
  description = "Days before S3 lineage events transition to Glacier."
  type        = number
}

variable "audit_retention_days" {
  description = "Days to retain gateway audit events in S3."
  type        = number
}
