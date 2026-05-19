#!/bin/bash
# Creates the core Mzigo topics in local Kafka.
#
# Topic design notes:
# - mzigo.contracts.events: low-volume, high-retention. Contract changes are
#   infrequent but must survive for audit. 30-day retention.
# - mzigo.lineage.events: medium-volume. One event per message batch produced.
#   7-day retention matches our lineage query window.
# - mzigo.gateway.violations: low-volume. Every contract violation lands here
#   for alerting and producer feedback. 14-day retention.
# - mzigo.gateway.audit: high-volume in busy environments. Sampling should be
#   applied upstream before writing here. 3-day retention locally.
#
# Partitions are set low for local dev. Production values live in Terraform.

set -euo pipefail

BOOTSTRAP="kafka:29092"

create_topic() {
  local topic=$1
  local partitions=$2
  local retention_ms=$3

  kafka-topics \
    --bootstrap-server "$BOOTSTRAP" \
    --create \
    --if-not-exists \
    --topic "$topic" \
    --partitions "$partitions" \
    --replication-factor 1 \
    --config "retention.ms=$retention_ms"

  echo "topic ready: $topic (partitions=$partitions, retention=${retention_ms}ms)"
}

echo "waiting for kafka to be available..."
sleep 5

create_topic "mzigo.contracts.events"   2  2592000000   # 30 days
create_topic "mzigo.lineage.events"     4  604800000    # 7 days
create_topic "mzigo.gateway.violations" 2  1209600000   # 14 days
create_topic "mzigo.gateway.audit"      4  259200000    # 3 days

echo "all topics created"
