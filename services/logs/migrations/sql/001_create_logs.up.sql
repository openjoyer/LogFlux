CREATE TABLE IF NOT EXISTS logflux.logs
(
    project_id   String,
    event_id     UUID,
    occurred_at  DateTime64(3, 'UTC'),
    received_at  DateTime64(3, 'UTC'),

    service      LowCardinality(String),
    environment  LowCardinality(String),
    level        LowCardinality(String),

    message      String,
    trace_id     String DEFAULT '',
    attributes   String DEFAULT '{}',
    fingerprint  String DEFAULT ''
)
    ENGINE = ReplacingMergeTree()
    PARTITION BY toDate(occurred_at)
    ORDER BY (project_id, occurred_at, event_id)
    TTL toDateTime(occurred_at) + INTERVAL 7 DAY;
