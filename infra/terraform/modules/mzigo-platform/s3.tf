resource "aws_kms_key" "s3" {
  description             = "Mzigo S3 bucket encryption"
  deletion_window_in_days = 14
  enable_key_rotation     = true

  tags = {
    Name = "${var.name_prefix}-s3-kms"
  }
}

# Lineage events bucket: long-term retention for OpenLineage events.
# The lineage worker writes events here after emitting to Marquez.
# This is the audit-grade historical record; Marquez is the query layer.
resource "aws_s3_bucket" "lineage" {
  bucket = "${var.name_prefix}-lineage-events"

  tags = {
    Name    = "${var.name_prefix}-lineage-events"
    Purpose = "OpenLineage event retention"
  }
}

resource "aws_s3_bucket_versioning" "lineage" {
  bucket = aws_s3_bucket.lineage.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "lineage" {
  bucket = aws_s3_bucket.lineage.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.s3.arn
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_public_access_block" "lineage" {
  bucket                  = aws_s3_bucket.lineage.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_lifecycle_configuration" "lineage" {
  bucket = aws_s3_bucket.lineage.id

  rule {
    id     = "lineage-retention"
    status = "Enabled"

    transition {
      days          = var.lineage_retention_days
      storage_class = "GLACIER"
    }

    expiration {
      # Permanent deletion after 5 years. Adjust to match your
      # data governance policy.
      days = 1825
    }
  }
}

# Gateway audit bucket: every accepted message is logged here.
# High volume in busy environments. Lifecycle to Glacier after 90 days
# keeps costs manageable while meeting compliance retention requirements.
resource "aws_s3_bucket" "audit" {
  bucket = "${var.name_prefix}-gateway-audit"

  tags = {
    Name    = "${var.name_prefix}-gateway-audit"
    Purpose = "Gateway message audit log retention"
  }
}

resource "aws_s3_bucket_versioning" "audit" {
  bucket = aws_s3_bucket.audit.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "audit" {
  bucket = aws_s3_bucket.audit.id
  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.s3.arn
    }
    bucket_key_enabled = true
  }
}

resource "aws_s3_bucket_public_access_block" "audit" {
  bucket                  = aws_s3_bucket.audit.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_lifecycle_configuration" "audit" {
  bucket = aws_s3_bucket.audit.id

  rule {
    id     = "audit-retention"
    status = "Enabled"

    transition {
      days          = 90
      storage_class = "GLACIER"
    }

    expiration {
      days = var.audit_retention_days
    }
  }
}
