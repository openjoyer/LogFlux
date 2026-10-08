#!/bin/sh
set -eu

KAFKA_TOPICS_BIN="${KAFKA_TOPICS_BIN:-/opt/kafka/bin/kafka-topics.sh}"
KAFKA_BOOTSTRAP_SERVERS="${KAFKA_BOOTSTRAP_SERVERS:-kafka:19092}"

KAFKA_REPLICATION_FACTOR="${KAFKA_REPLICATION_FACTOR:-1}"
KAFKA_MIN_INSYNC_REPLICAS="${KAFKA_MIN_INSYNC_REPLICAS:-1}"
KAFKA_TOPIC="${KAFKA_TOPIC:-logflux.logs.accepted.v1}"
KAFKA_DLQ_TOPIC="${KAFKA_DLQ_TOPIC:-logflux.processing.dlq.v1}"
KAFKA_LOG_PARTITIONS="${KAFKA_LOG_PARTITIONS:-6}"
KAFKA_DLQ_PARTITIONS="${KAFKA_DLQ_PARTITIONS:-3}"
KAFKA_LOG_RETENTION_MS="${KAFKA_LOG_RETENTION_MS:-259200000}" # 3 days
KAFKA_DLQ_RETENTION_MS="${KAFKA_DLQ_RETENTION_MS:-604800000}" # 7 days

positive_integer() {
    case "$2" in
        ''|*[!0-9]*|0*)
            printf 'Invalid %s: expected a positive integer without leading zeros\n' "$1" >&2
            exit 1
            ;;
    esac
}

positive_integer KAFKA_REPLICATION_FACTOR "$KAFKA_REPLICATION_FACTOR"
positive_integer KAFKA_MIN_INSYNC_REPLICAS "$KAFKA_MIN_INSYNC_REPLICAS"
positive_integer KAFKA_LOG_PARTITIONS "$KAFKA_LOG_PARTITIONS"
positive_integer KAFKA_DLQ_PARTITIONS "$KAFKA_DLQ_PARTITIONS"
positive_integer KAFKA_LOG_RETENTION_MS "$KAFKA_LOG_RETENTION_MS"
positive_integer KAFKA_DLQ_RETENTION_MS "$KAFKA_DLQ_RETENTION_MS"

if [ "$KAFKA_MIN_INSYNC_REPLICAS" -gt "$KAFKA_REPLICATION_FACTOR" ]; then
    printf 'min.insync.replicas must not exceed replication factor\n' >&2
    exit 1
fi
if [ "$KAFKA_TOPIC" = "$KAFKA_DLQ_TOPIC" ]; then
    printf 'Log and DLQ topics must have different names\n' >&2
    exit 1
fi

create_topic() {
    "$KAFKA_TOPICS_BIN" \
        --bootstrap-server "$KAFKA_BOOTSTRAP_SERVERS" \
        --create \
        --if-not-exists \
        --topic "$1" \
        --partitions "$2" \
        --replication-factor "$KAFKA_REPLICATION_FACTOR" \
        --config "min.insync.replicas=$KAFKA_MIN_INSYNC_REPLICAS" \
        --config "unclean.leader.election.enable=false" \
        --config "cleanup.policy=delete" \
        --config "retention.ms=$3"
}

create_topic "$KAFKA_TOPIC" "$KAFKA_LOG_PARTITIONS" "$KAFKA_LOG_RETENTION_MS"

create_topic "$KAFKA_DLQ_TOPIC" "$KAFKA_DLQ_PARTITIONS" "$KAFKA_DLQ_RETENTION_MS"