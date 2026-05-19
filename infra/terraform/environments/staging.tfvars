environment = "staging"
aws_region  = "us-east-1"

availability_zones = ["us-east-1a", "us-east-1b", "us-east-1c"]
vpc_cidr           = "10.1.0.0/16"

# Smaller brokers for staging. Scale up instance type and count for production.
msk_kafka_version        = "3.6.0"
msk_broker_instance_type = "kafka.t3.small"
msk_broker_count         = 3
msk_broker_storage_gb    = 100

# Single-AZ RDS in staging to reduce cost.
# multi_az is automatically false when environment != "production".
rds_instance_class       = "db.t3.micro"
rds_allocated_storage_gb = 20
rds_postgres_version     = "16.1"

eks_cluster_name = "mzigo-staging"

lineage_retention_days = 30
audit_retention_days   = 90
