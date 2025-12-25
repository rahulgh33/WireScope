#!/bin/bash
# Quick AWS deployment script

set -e

REGION=${AWS_REGION:-us-east-1}
CLUSTER_NAME=${CLUSTER_NAME:-wirescope}

echo "Deploying to AWS region: $REGION"

# Create RDS PostgreSQL
aws rds create-db-instance \
  --db-instance-identifier wirescope-db \
  --db-instance-class db.t3.micro \
  --engine postgres \
  --master-username wirescope \
  --master-user-password "${DB_PASSWORD}" \
  --allocated-storage 20 \
  --region "$REGION"

# Create ECS cluster
aws ecs create-cluster --cluster-name "$CLUSTER_NAME" --region "$REGION"

# Deploy services (simplified - use proper task definitions in production)
echo "Cluster created. Configure task definitions and services next."
echo "See AWS ECS/Fargate docs for full setup."
