#!/usr/bin/env bash
set -euo pipefail

kafka_bin_dir=${KAFKA_BIN_DIR:-/opt/kafka/bin}
client_args=(--bootstrap-server "${KAFKA_BOOTSTRAP_SERVERS:-localhost:9092}")
if [[ -n ${KAFKA_ADMIN_CONFIG:-} ]]; then
  client_args+=(--command-config "$KAFKA_ADMIN_CONFIG")
fi

"$kafka_bin_dir/kafka-topics.sh" "${client_args[@]}" \
  --create \
  --if-not-exists \
  --topic notification.otp.v1 \
  --partitions 3 \
  --replication-factor 3 \
  --config min.insync.replicas=2 \
  --config cleanup.policy=delete \
  --config retention.ms=604800000

"$kafka_bin_dir/kafka-topics.sh" "${client_args[@]}" \
    --create \
    --if-not-exists \
    --topic notification.request.v1 \
    --partitions 3 \
    --replication-factor 3 \
    --config min.insync.replicas=2 \
    --config cleanup.policy=delete \
    --config retention.ms=604800000
