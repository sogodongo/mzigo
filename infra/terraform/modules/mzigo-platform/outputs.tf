output "msk_bootstrap_brokers_tls" {
  description = "MSK TLS bootstrap broker endpoints."
  value       = aws_msk_cluster.main.bootstrap_brokers_tls
}

output "msk_schema_registry_url" {
  description = "MSK cluster ARN. Used for IAM auth policy resource references."
  value       = aws_msk_cluster.main.arn
}

output "rds_endpoint" {
  description = "RDS instance hostname."
  value       = aws_db_instance.contracts.address
}

output "rds_port" {
  description = "RDS instance port."
  value       = aws_db_instance.contracts.port
}

output "rds_secret_arn" {
  description = "Secrets Manager ARN for the RDS password. Pass to contracts service IRSA policy."
  value       = aws_secretsmanager_secret.rds_password.arn
}

output "lineage_bucket_name" {
  description = "S3 bucket name for OpenLineage event retention."
  value       = aws_s3_bucket.lineage.id
}

output "lineage_bucket_arn" {
  description = "S3 bucket ARN for OpenLineage event retention."
  value       = aws_s3_bucket.lineage.arn
}

output "audit_bucket_name" {
  description = "S3 bucket name for gateway audit log retention."
  value       = aws_s3_bucket.audit.id
}

output "gateway_irsa_role_arn" {
  description = "IAM role ARN for the gateway Kubernetes ServiceAccount."
  value       = aws_iam_role.gateway.arn
}

output "contracts_irsa_role_arn" {
  description = "IAM role ARN for the contracts Kubernetes ServiceAccount."
  value       = aws_iam_role.contracts.arn
}

output "lineage_irsa_role_arn" {
  description = "IAM role ARN for the lineage worker Kubernetes ServiceAccount."
  value       = aws_iam_role.lineage.arn
}

output "masking_irsa_role_arn" {
  description = "IAM role ARN for the masking service Kubernetes ServiceAccount."
  value       = aws_iam_role.masking.arn
}

output "msk_security_group_id" {
  description = "Security group ID for MSK brokers."
  value       = aws_security_group.msk.id
}

output "rds_security_group_id" {
  description = "Security group ID for RDS."
  value       = aws_security_group.rds.id
}
