output "msk_bootstrap_brokers_tls" {
  description = "MSK broker connection string for TLS clients."
  value       = module.mzigo_platform.msk_bootstrap_brokers_tls
  sensitive   = false
}

output "msk_schema_registry_url" {
  description = "Glue Schema Registry URL (if using MSK with Glue SR)."
  value       = module.mzigo_platform.msk_schema_registry_url
  sensitive   = false
}

output "rds_endpoint" {
  description = "RDS instance endpoint for the contracts database."
  value       = module.mzigo_platform.rds_endpoint
  sensitive   = false
}

output "rds_port" {
  description = "RDS instance port."
  value       = module.mzigo_platform.rds_port
  sensitive   = false
}

output "lineage_bucket_name" {
  description = "S3 bucket for OpenLineage event retention."
  value       = module.mzigo_platform.lineage_bucket_name
  sensitive   = false
}

output "audit_bucket_name" {
  description = "S3 bucket for gateway audit log retention."
  value       = module.mzigo_platform.audit_bucket_name
  sensitive   = false
}

output "gateway_irsa_role_arn" {
  description = "IAM role ARN for the gateway service account (IRSA)."
  value       = module.mzigo_platform.gateway_irsa_role_arn
  sensitive   = false
}

output "contracts_irsa_role_arn" {
  description = "IAM role ARN for the contracts service account (IRSA)."
  value       = module.mzigo_platform.contracts_irsa_role_arn
  sensitive   = false
}

output "lineage_irsa_role_arn" {
  description = "IAM role ARN for the lineage worker service account (IRSA)."
  value       = module.mzigo_platform.lineage_irsa_role_arn
  sensitive   = false
}

output "vpc_id" {
  description = "VPC ID for the Mzigo deployment."
  value       = module.networking.vpc_id
  sensitive   = false
}
