# Kafka topics

Run from the repository root after all three brokers are ready:

```sh
docker compose exec -T kafka-1 bash < infra/kafka/create-topics.sh
```

Creates `otp.v1` with 3 partitions, 3 replicas and 7 days
of retention. Existing topics are skipped; their settings are not checked or
changed. Add another `kafka-topics.sh --create` command for each new topic.

Go topic names are declared in `pkg/kafkaevents/topics.go`. Keep this script
aligned with those constants. Auth checks the OTP topic's metadata with a
5-second timeout before starting its gRPC server. The check does not create
topics or send test messages; delivery errors must still be handled at runtime.

For production, run the same script as a deployment step with the Kafka CLI
installed and credentials supplied through an external client properties file:

```sh
KAFKA_BOOTSTRAP_SERVERS=broker-1:9092,broker-2:9092,broker-3:9092 \
KAFKA_ADMIN_CONFIG=/run/secrets/kafka-admin.properties \
bash infra/kafka/create-topics.sh
```

`KAFKA_BIN_DIR` defaults to `/opt/kafka/bin`. Broker auto-creation is disabled
in Compose; already running brokers receive that change when recreated.
