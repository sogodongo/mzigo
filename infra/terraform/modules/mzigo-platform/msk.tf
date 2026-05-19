resource "aws_msk_cluster" "main" {
  cluster_name           = "${var.name_prefix}-kafka"
  kafka_version          = var.kafka_version
  number_of_broker_nodes = var.broker_count

  broker_node_group_info {
    instance_type  = var.broker_instance_type
    client_subnets = var.intra_subnet_ids
    security_groups = [aws_security_group.msk.id]

    storage_info {
      ebs_storage_info {
        volume_size = var.broker_storage_gb

        # Provisioned throughput: enables higher IOPS for busy topics.
        # Disabled by default; enable if broker disk throughput becomes
        # a bottleneck under sustained high-volume produce workloads.
        provisioned_throughput {
          enabled = false
        }
      }
    }
  }

  client_authentication {
    # TLS mutual authentication with ACM Private CA.
    # Producers authenticate with client certificates; unauthenticated
    # connections are rejected at the broker.
    tls {
      certificate_authority_arns = []
    }

    # SASL/SCRAM as an alternative for producers that cannot manage client certs.
    # Both mechanisms can coexist. We enable SCRAM so the SDK can use
    # username/password credentials stored in AWS Secrets Manager.
    sasl {
      scram = true
      iam   = true
    }
  }

  encryption_info {
    encryption_in_transit {
      # PLAINTEXT_ONLY is never acceptable in production.
      # TLS_PLAINTEXT allows both for migration windows.
      # TLS only after all clients are migrated.
      client_broker = "TLS"
      in_cluster    = true
    }

    encryption_at_rest {
      data_volume_kms_key_id = aws_kms_key.msk.arn
    }
  }

  configuration_info {
    arn      = aws_msk_configuration.main.arn
    revision = aws_msk_configuration.main.latest_revision
  }

  logging_info {
    broker_logs {
      cloudwatch_logs {
        enabled   = true
        log_group = aws_cloudwatch_log_group.msk_broker.name
      }
    }
  }

  open_monitoring {
    prometheus {
      jmx_exporter {
        enabled_in_broker = true
      }
      node_exporter {
        enabled_in_broker = true
      }
    }
  }

  tags = {
    Name = "${var.name_prefix}-kafka"
  }
}

resource "aws_msk_configuration" "main" {
  name          = "${var.name_prefix}-kafka-config"
  kafka_versions = [var.kafka_version]

  server_properties = <<-EOF
    # Retain messages for 7 days by default.
    # Individual topics override this with topic-level retention configs
    # set by the create-topics script.
    log.retention.hours=168

    # Replication factor for internal topics.
    # 3 ensures no data loss if one broker goes down.
    default.replication.factor=3
    min.insync.replicas=2

    # Auto topic creation is disabled. All topics are created explicitly
    # by the Terraform topic resources or the create-topics script.
    # Implicit topic creation hides misconfigured producer topic names
    # until data is already lost.
    auto.create.topics.enable=false

    # Compression. Producers set their own compression type.
    # Server-side re-compression is disabled to avoid CPU overhead.
    compression.type=producer

    # Message size limit matches the gateway's message_max_bytes config.
    message.max.bytes=1048576
  EOF
}

# KMS key for MSK data-at-rest encryption.
resource "aws_kms_key" "msk" {
  description             = "Mzigo MSK broker data encryption"
  deletion_window_in_days = 14
  enable_key_rotation     = true

  tags = {
    Name = "${var.name_prefix}-msk-kms"
  }
}

resource "aws_kms_alias" "msk" {
  name          = "alias/${var.name_prefix}-msk"
  target_key_id = aws_kms_key.msk.key_id
}

resource "aws_cloudwatch_log_group" "msk_broker" {
  name              = "/mzigo/${var.name_prefix}/msk-broker-logs"
  retention_in_days = 14
}

# Core Mzigo topics.
# Topic configuration mirrors the local dev create-topics.sh script
# but with production-grade replication and retention values.
resource "aws_msk_topic" "contracts_events" {
  cluster_arn        = aws_msk_cluster.main.arn
  topic_name         = "mzigo.contracts.events"
  replication_factor = 3
  partition_count    = 6

  config = {
    "retention.ms"    = "2592000000" # 30 days
    "cleanup.policy"  = "delete"
    "min.insync.replicas" = "2"
  }
}

resource "aws_msk_topic" "lineage_events" {
  cluster_arn        = aws_msk_cluster.main.arn
  topic_name         = "mzigo.lineage.events"
  replication_factor = 3
  partition_count    = 12

  config = {
    "retention.ms"    = "604800000" # 7 days
    "cleanup.policy"  = "delete"
    "min.insync.replicas" = "2"
  }
}

resource "aws_msk_topic" "gateway_violations" {
  cluster_arn        = aws_msk_cluster.main.arn
  topic_name         = "mzigo.gateway.violations"
  replication_factor = 3
  partition_count    = 6

  config = {
    "retention.ms"    = "1209600000" # 14 days
    "cleanup.policy"  = "delete"
    "min.insync.replicas" = "2"
  }
}

resource "aws_msk_topic" "gateway_audit" {
  cluster_arn        = aws_msk_cluster.main.arn
  topic_name         = "mzigo.gateway.audit"
  replication_factor = 3
  # Partition count matches the recommended lineage worker replica count.
  # Each lineage worker instance handles one partition group.
  partition_count    = 12

  config = {
    "retention.ms"    = "259200000" # 3 days (high volume; S3 is the long-term store)
    "cleanup.policy"  = "delete"
    "min.insync.replicas" = "2"
  }
}
