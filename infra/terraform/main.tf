locals {
  name_prefix = "mzigo-${var.environment}"
}

module "networking" {
  source = "./modules/networking"

  name_prefix        = local.name_prefix
  vpc_cidr           = var.vpc_cidr
  availability_zones = var.availability_zones
}

module "mzigo_platform" {
  source = "./modules/mzigo-platform"

  name_prefix        = local.name_prefix
  environment        = var.environment
  aws_region         = var.aws_region
  eks_cluster_name   = var.eks_cluster_name

  # Networking inputs from the networking module.
  # Platform resources never reference VPC or subnet IDs directly;
  # they receive them from the networking module. This enforces the
  # module boundary: a network team can swap the networking module
  # without touching platform resource definitions.
  vpc_id              = module.networking.vpc_id
  private_subnet_ids  = module.networking.private_subnet_ids
  intra_subnet_ids    = module.networking.intra_subnet_ids

  # MSK
  kafka_version        = var.msk_kafka_version
  broker_instance_type = var.msk_broker_instance_type
  broker_count         = var.msk_broker_count
  broker_storage_gb    = var.msk_broker_storage_gb

  # RDS
  rds_instance_class       = var.rds_instance_class
  rds_allocated_storage_gb = var.rds_allocated_storage_gb
  rds_postgres_version     = var.rds_postgres_version

  # S3 retention
  lineage_retention_days = var.lineage_retention_days
  audit_retention_days   = var.audit_retention_days
}
