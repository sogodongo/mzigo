resource "random_password" "rds" {
  length           = 32
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

# Store the generated password in Secrets Manager.
# The contracts service reads it from here via IRSA, not from a hardcoded value.
resource "aws_secretsmanager_secret" "rds_password" {
  name                    = "${var.name_prefix}/rds/contracts-password"
  description             = "Mzigo contracts database password"
  recovery_window_in_days = 14

  tags = {
    Name = "${var.name_prefix}-rds-password"
  }
}

resource "aws_secretsmanager_secret_version" "rds_password" {
  secret_id = aws_secretsmanager_secret.rds_password.id
  secret_string = jsonencode({
    username = "mzigo"
    password = random_password.rds.result
    host     = aws_db_instance.contracts.address
    port     = aws_db_instance.contracts.port
    dbname   = "mzigo"
    url      = "postgres://mzigo:${random_password.rds.result}@${aws_db_instance.contracts.address}:${aws_db_instance.contracts.port}/mzigo"
  })
}

resource "aws_db_subnet_group" "main" {
  name        = "${var.name_prefix}-db-subnet-group"
  description = "Mzigo RDS subnet group (intra subnets, no internet routing)"
  subnet_ids  = var.intra_subnet_ids

  tags = {
    Name = "${var.name_prefix}-db-subnet-group"
  }
}

resource "aws_kms_key" "rds" {
  description             = "Mzigo RDS storage encryption"
  deletion_window_in_days = 14
  enable_key_rotation     = true

  tags = {
    Name = "${var.name_prefix}-rds-kms"
  }
}

resource "aws_db_instance" "contracts" {
  identifier = "${var.name_prefix}-contracts"

  engine         = "postgres"
  engine_version = var.rds_postgres_version
  instance_class = var.rds_instance_class

  allocated_storage     = var.rds_allocated_storage_gb
  max_allocated_storage = var.rds_allocated_storage_gb * 4
  storage_type          = "gp3"
  storage_encrypted     = true
  kms_key_id            = aws_kms_key.rds.arn

  db_name  = "mzigo"
  username = "mzigo"
  password = random_password.rds.result

  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.rds.id]
  publicly_accessible    = false

  # Multi-AZ provides automatic failover to a standby replica.
  # Disable in staging to reduce cost; enable in production.
  multi_az = var.environment == "production"

  # Automated backups retained for 14 days with PITR.
  backup_retention_period = 14
  backup_window           = "03:00-04:00"
  maintenance_window      = "sun:04:00-sun:05:00"

  # Prevent accidental deletion in production.
  # Terraform will error rather than destroy the database unless
  # this is explicitly set to false in a targeted apply.
  deletion_protection = var.environment == "production"

  skip_final_snapshot       = var.environment != "production"
  final_snapshot_identifier = "${var.name_prefix}-contracts-final-snapshot"

  performance_insights_enabled          = true
  performance_insights_retention_period = 7

  enabled_cloudwatch_logs_exports = ["postgresql", "upgrade"]

  tags = {
    Name = "${var.name_prefix}-contracts-db"
  }
}
