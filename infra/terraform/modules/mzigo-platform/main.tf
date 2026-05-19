# Data source: current AWS account ID and partition.
# Used in IAM policy ARN construction.
data "aws_caller_identity" "current" {}
data "aws_partition" "current" {}

# Data source: EKS cluster OIDC issuer for IRSA trust policies.
# IRSA (IAM Roles for Service Accounts) allows pods to assume IAM roles
# without node-level instance profiles. Each Mzigo service gets its own
# role with the minimum permissions it needs.
data "aws_eks_cluster" "main" {
  name = var.eks_cluster_name
}

locals {
  account_id   = data.aws_caller_identity.current.account_id
  partition    = data.aws_partition.current.partition
  # Strip the https:// prefix from the OIDC issuer URL for trust policy construction.
  oidc_issuer  = trimprefix(data.aws_eks_cluster.main.identity[0].oidc[0].issuer, "https://")
  oidc_arn     = "arn:${local.partition}:iam::${local.account_id}:oidc-provider/${local.oidc_issuer}"
}

# Security group: MSK brokers.
# Allows inbound Kafka (9094 TLS) and Schema Registry (9096) only from
# the private subnets where EKS nodes run. Brokers have no public access.
resource "aws_security_group" "msk" {
  name        = "${var.name_prefix}-msk"
  description = "Mzigo MSK broker access from EKS workloads"
  vpc_id      = var.vpc_id

  ingress {
    description = "Kafka TLS from private subnets"
    from_port   = 9094
    to_port     = 9094
    protocol    = "tcp"
    cidr_blocks = [for s in data.aws_subnet.private : s.cidr_block]
  }

  ingress {
    description = "Schema Registry from private subnets"
    from_port   = 9096
    to_port     = 9096
    protocol    = "tcp"
    cidr_blocks = [for s in data.aws_subnet.private : s.cidr_block]
  }

  egress {
    description = "Allow all outbound within VPC"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.name_prefix}-msk-sg"
  }
}

# Security group: RDS Postgres.
# Accepts connections only from EKS pods in private subnets.
resource "aws_security_group" "rds" {
  name        = "${var.name_prefix}-rds"
  description = "Mzigo RDS access from EKS workloads"
  vpc_id      = var.vpc_id

  ingress {
    description = "Postgres from private subnets"
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = [for s in data.aws_subnet.private : s.cidr_block]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.name_prefix}-rds-sg"
  }
}

# Data sources to fetch the private subnet CIDR blocks for security group rules.
data "aws_subnet" "private" {
  for_each = toset(var.private_subnet_ids)
  id       = each.value
}
