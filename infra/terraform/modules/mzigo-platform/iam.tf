# IAM roles using IRSA (IAM Roles for Service Accounts).
#
# IRSA binds an IAM role to a specific Kubernetes ServiceAccount in a specific
# namespace. The pod assumes the role via the projected service account token,
# not via a node instance profile. This means:
# - Pods get only the permissions they need
# - Compromising one pod does not grant IAM access to other services
# - No long-lived access keys stored anywhere
#
# The trust policy restricts role assumption to the exact ServiceAccount
# in the exact namespace. Wildcard subject matching is not used.

locals {
  # Kubernetes namespace where Mzigo is deployed.
  # This must match the --namespace flag in the helm install command.
  k8s_namespace = "mzigo"
}

# Gateway IAM role.
# The gateway needs to produce to MSK (handled by Kafka client auth, not IAM)
# and write audit events to S3.
resource "aws_iam_role" "gateway" {
  name = "${var.name_prefix}-gateway"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Federated = local.oidc_arn
      }
      Action = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "${local.oidc_issuer}:sub" = "system:serviceaccount:${local.k8s_namespace}:${var.name_prefix}-gateway"
          "${local.oidc_issuer}:aud" = "sts.amazonaws.com"
        }
      }
    }]
  })
}

resource "aws_iam_role_policy" "gateway_s3" {
  name = "${var.name_prefix}-gateway-s3"
  role = aws_iam_role.gateway.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = ["s3:PutObject"]
        Resource = "${aws_s3_bucket.audit.arn}/*"
      },
      {
        Effect = "Allow"
        Action = ["kms:GenerateDataKey", "kms:Decrypt"]
        Resource = aws_kms_key.s3.arn
      }
    ]
  })
}

# Contracts IAM role.
# Needs to read from Secrets Manager (RDS password).
resource "aws_iam_role" "contracts" {
  name = "${var.name_prefix}-contracts"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Federated = local.oidc_arn
      }
      Action = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "${local.oidc_issuer}:sub" = "system:serviceaccount:${local.k8s_namespace}:${var.name_prefix}-contracts"
          "${local.oidc_issuer}:aud" = "sts.amazonaws.com"
        }
      }
    }]
  })
}

resource "aws_iam_role_policy" "contracts_secrets" {
  name = "${var.name_prefix}-contracts-secrets"
  role = aws_iam_role.contracts.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = ["secretsmanager:GetSecretValue"]
        Resource = aws_secretsmanager_secret.rds_password.arn
      },
      {
        Effect   = "Allow"
        Action   = ["kms:Decrypt"]
        Resource = aws_kms_key.rds.arn
      }
    ]
  })
}

# Lineage worker IAM role.
# Needs MSK consumer access (handled by SASL/IAM) and S3 write for event archival.
resource "aws_iam_role" "lineage" {
  name = "${var.name_prefix}-lineage"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Federated = local.oidc_arn
      }
      Action = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "${local.oidc_issuer}:sub" = "system:serviceaccount:${local.k8s_namespace}:${var.name_prefix}-lineage"
          "${local.oidc_issuer}:aud" = "sts.amazonaws.com"
        }
      }
    }]
  })
}

resource "aws_iam_role_policy" "lineage" {
  name = "${var.name_prefix}-lineage"
  role = aws_iam_role.lineage.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        # MSK IAM auth: allows the lineage worker to consume from Mzigo topics.
        Effect = "Allow"
        Action = [
          "kafka-cluster:Connect",
          "kafka-cluster:DescribeGroup",
          "kafka-cluster:AlterGroup",
          "kafka-cluster:DescribeTopic",
          "kafka-cluster:ReadData",
          "kafka-cluster:DescribeClusterDynamicConfiguration",
        ]
        Resource = [
          aws_msk_cluster.main.arn,
          "${aws_msk_cluster.main.arn}/topic/mzigo.gateway.audit",
          "${aws_msk_cluster.main.arn}/topic/mzigo.gateway.violations",
          "${aws_msk_cluster.main.arn}/group/mzigo-lineage-worker",
        ]
      },
      {
        Effect   = "Allow"
        Action   = ["s3:PutObject"]
        Resource = "${aws_s3_bucket.lineage.arn}/*"
      },
      {
        Effect   = "Allow"
        Action   = ["kms:GenerateDataKey", "kms:Decrypt"]
        Resource = aws_kms_key.s3.arn
      }
    ]
  })
}

# Masking service IAM role.
# The masking service needs access to the KMS key for tokenization
# in environments where the HMAC key is stored in KMS rather than
# a static Kubernetes Secret.
resource "aws_iam_role" "masking" {
  name = "${var.name_prefix}-masking"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Federated = local.oidc_arn
      }
      Action = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "${local.oidc_issuer}:sub" = "system:serviceaccount:${local.k8s_namespace}:${var.name_prefix}-masking"
          "${local.oidc_issuer}:aud" = "sts.amazonaws.com"
        }
      }
    }]
  })
}

resource "aws_iam_role_policy" "masking_secrets" {
  name = "${var.name_prefix}-masking-secrets"
  role = aws_iam_role.masking.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["secretsmanager:GetSecretValue"]
      Resource = "arn:${local.partition}:secretsmanager:${var.aws_region}:${local.account_id}:secret:${var.name_prefix}/masking/*"
    }]
  })
}
