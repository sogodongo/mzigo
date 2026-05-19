variable "aws_region" {
  description = "AWS region for all resources."
  type        = string
  default     = "us-east-1"
}

variable "environment" {
  description = "Deployment environment. Used in resource names and tags."
  type        = string
  validation {
    condition     = contains(["staging", "production"], var.environment)
    error_message = "environment must be 'staging' or 'production'."
  }
}

variable "vpc_cidr" {
  description = "CIDR block for the Mzigo VPC."
  type        = string
  default     = "10.0.0.0/16"
}

variable "availability_zones" {
  description = "AZs to deploy into. Three AZs required for MSK high availability."
  type        = list(string)
  default     = ["us-east-1a", "us-east-1b", "us-east-1c"]

  validation {
    condition     = length(var.availability_zones) >= 3
    error_message = "At least three availability zones are required for MSK."
  }
}

variable "msk_kafka_version" {
  description = "Apache Kafka version for the MSK cluster."
  type        = string
  default     = "3.6.0"
}

variable "msk_broker_instance_type" {
  description = "EC2 instance type for MSK brokers."
  type        = string
  default     = "kafka.m5.large"
}

variable "msk_broker_count" {
  description = "Number of MSK broker nodes. Must be a multiple of the AZ count."
  type        = number
  default     = 3

  validation {
    condition     = var.msk_broker_count >= 3 && var.msk_broker_count % 3 == 0
    error_message = "msk_broker_count must be >= 3 and a multiple of 3 (one per AZ)."
  }
}

variable "msk_broker_storage_gb" {
  description = "EBS storage per MSK broker in GB."
  type        = number
  default     = 500
}

variable "rds_instance_class" {
  description = "RDS instance class for the contracts database."
  type        = string
  default     = "db.t3.medium"
}

variable "rds_allocated_storage_gb" {
  description = "Initial RDS storage in GB. Autoscaling handles growth."
  type        = number
  default     = 50
}

variable "rds_postgres_version" {
  description = "PostgreSQL engine version."
  type        = string
  default     = "16.1"
}

variable "eks_cluster_name" {
  description = "Name of the existing EKS cluster Mzigo is deployed into."
  type        = string
}

variable "lineage_retention_days" {
  description = "Days to retain lineage events in S3 before transitioning to Glacier."
  type        = number
  default     = 90
}

variable "audit_retention_days" {
  description = "Days to retain gateway audit events in S3. Compliance-driven."
  type        = number
  default     = 365
}
