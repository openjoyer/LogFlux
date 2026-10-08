package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadLocalConfigExpandsEnvironment(t *testing.T) {
	password := "secret: # with special characters\nand newline"
	for key, value := range map[string]string{
		"LOGS_HTTP_ADDRESS":        "0.0.0.0",
		"LOGS_HTTP_PORT":           "8080",
		"KAFKA_BROKER":             "kafka:19092",
		"KAFKA_TOPIC":              "logflux.logs.accepted",
		"LOGS_KAFKA_BATCH_TIMEOUT": "250ms",
		"LOGS_KAFKA_WRITE_TIMEOUT": "10s",
		"CLICKHOUSE_ADDRESS":       "clickhouse:9000",
		"LOGS_CLICKHOUSE_DB":       "logflux",
		"LOGS_CLICKHOUSE_USERNAME": "logflux",
		"CLICKHOUSE_PASSWORD":      password,
	} {
		t.Setenv(key, value)
	}

	cfg, err := Load(filepath.Join("..", "..", "config", "local.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Kafka.Brokers) != 1 || cfg.Kafka.Brokers[0] != "kafka:19092" ||
		cfg.Kafka.Topic != "logflux.logs.accepted" || cfg.Kafka.BatchTimeout != 250*time.Millisecond {
		t.Fatalf("Kafka configuration was decoded incorrectly: %+v", cfg.Kafka)
	}
	if cfg.ClickHouse.Database != "logflux" || cfg.ClickHouse.Password != password ||
		len(cfg.ClickHouse.Addresses) != 1 || cfg.ClickHouse.Addresses[0] != "clickhouse:9000" {
		t.Fatalf("ClickHouse configuration was decoded incorrectly: %+v", cfg.ClickHouse)
	}
	if cfg.Batch.MaxEvents != 5000 || cfg.Batch.FlushInterval != time.Second ||
		cfg.Writer.MaxAttempts != 5 || cfg.ShutdownTimeout != 30*time.Second {
		t.Fatalf("batch or writer settings were decoded incorrectly: %+v", cfg)
	}
}

func TestLoadReportsMissingEnvironmentVariable(t *testing.T) {
	const key = "LOGFLUX_LOGS_CONFIG_TEST_MISSING"
	t.Setenv(key, "")
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}

	path := writeConfig(t, "kafka:\n  topic: ${"+key+"}\n")
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), key) {
		t.Fatalf("expected error naming missing variable, got %v", err)
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	path := writeConfig(t, "batch:\n  flush_interval: sometime\n")
	if _, err := Load(path); err == nil {
		t.Fatal("expected invalid flush interval to fail decoding")
	}
}

func writeConfig(t *testing.T, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
