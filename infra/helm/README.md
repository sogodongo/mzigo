# Mzigo Helm Charts

Kubernetes deployment packaging for the Mzigo platform.

## Structure

```
infra/helm/
├── charts/          Individual service charts
│   ├── gateway/
│   ├── contracts/
│   ├── lineage/
│   ├── analyzer/
│   └── masking/
└── mzigo/           Umbrella chart deploying the full platform
```

## Prerequisites

- Kubernetes 1.25+
- Helm 3.12+
- Prometheus Operator (for ServiceMonitor resources)
- A running Kafka cluster and Schema Registry
- A Postgres database for the contracts service
- A Marquez instance (or deploy one separately)

## Quick Install

```bash
# From the repository root
cd infra/helm

# Update subchart dependencies
helm dependency update mzigo/

# Create namespace
kubectl create namespace mzigo

# Create required secrets (see Secrets section below)
# ...

# Install
helm install mzigo ./mzigo \
  --namespace mzigo \
  --values mzigo/values.production.yaml.example
```

## Secrets

Each service expects a Kubernetes Secret containing its sensitive configuration.
Create these before installing:

```bash
# Gateway
kubectl create secret generic mzigo-gateway-secrets \
  --namespace mzigo \
  --from-literal=MZIGO_GATEWAY_KAFKA_BOOTSTRAP_SERVERS="broker1:9092,broker2:9092" \
  --from-literal=MZIGO_GATEWAY_CONTRACTS_SERVICE_URL="http://mzigo-contracts:8081" \
  --from-literal=MZIGO_GATEWAY_OTEL_ENDPOINT="otel-collector:4317"

# Contracts
kubectl create secret generic mzigo-contracts-secrets \
  --namespace mzigo \
  --from-literal=DATABASE_URL="postgres://mzigo:password@postgres:5432/mzigo"

# Lineage
kubectl create secret generic mzigo-lineage-secrets \
  --namespace mzigo \
  --from-literal=MZIGO_LINEAGE_KAFKA_BOOTSTRAP_SERVERS="broker1:9092,broker2:9092" \
  --from-literal=MZIGO_LINEAGE_OPENLINEAGE_URL="http://marquez:5000" \
  --from-literal=MZIGO_LINEAGE_DATABASE_URL="postgres://mzigo:password@postgres:5432/mzigo"

# Analyzer
kubectl create secret generic mzigo-analyzer-secrets \
  --namespace mzigo \
  --from-literal=MZIGO_ANALYZER_MARQUEZ_URL="http://marquez:5000" \
  --from-literal=MZIGO_ANALYZER_DATABASE_URL="postgres://mzigo:password@postgres:5432/mzigo"

# Masking - HMAC key must be a cryptographically random secret
kubectl create secret generic mzigo-masking-secrets \
  --namespace mzigo \
  --from-literal=MZIGO_MASKING_TOKENIZATION_KEY="$(openssl rand -hex 32)"
```

## Upgrading

```bash
helm upgrade mzigo ./mzigo \
  --namespace mzigo \
  --values mzigo/values.production.yaml \
  --atomic \
  --timeout 5m
```

`--atomic` rolls back automatically if any pod fails to become ready.

## Deploying a Single Service

```bash
helm install mzigo-gateway ./charts/gateway \
  --namespace mzigo \
  --set extraEnvFrom[0].secretRef.name=mzigo-gateway-secrets
```

## Scaling

Each service scales independently. To scale the gateway:

```bash
kubectl scale deployment mzigo-gateway --replicas=5 -n mzigo
```

Or update the values and upgrade:

```bash
helm upgrade mzigo ./mzigo --namespace mzigo \
  --set mzigo-gateway.replicaCount=5
```

## Uninstall

```bash
helm uninstall mzigo --namespace mzigo
kubectl delete namespace mzigo
```
