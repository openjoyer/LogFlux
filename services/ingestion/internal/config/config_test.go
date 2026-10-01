package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadEnvironment(t *testing.T) {
	t.Setenv("TEST_CONFIG_DB", "2")
	t.Setenv("TEST_CONFIG_TIMEOUT", "5s")
	t.Setenv("TEST_CONFIG_PASSWORD", "secret: # value\nwith newline")
	t.Setenv("TEST_CONFIG_BROKER", "localhost:9092")

	path := filepath.Join(t.TempDir(), "config.yaml")
	data := "redis:\n  db: ${TEST_CONFIG_DB}\n  password: ${TEST_CONFIG_PASSWORD}\nkafka:\n  write_timeout: ${TEST_CONFIG_TIMEOUT}\n  brokers:\n    - ${TEST_CONFIG_BROKER}\n"

	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Redis.DB != 2 || cfg.Kafka.WriteTimeout != 5*time.Second {
		t.Fatal("environment values were not decoded into their target types")
	}
	if cfg.Redis.Password != os.Getenv("TEST_CONFIG_PASSWORD") {
		t.Fatal("password contents changed during substitution")
	}
	if len(cfg.Kafka.Brokers) != 1 || cfg.Kafka.Brokers[0] != "localhost:9092" {
		t.Fatal("broker substitution failed")
	}

	t.Setenv("TEST_CONFIG_PASSWORD", "")
	cfg, err = Load(path)
	if err != nil || cfg.Redis.Password != "" {
		t.Fatal("empty password should be accepted")
	}
}

func TestLoadMissingEnvironment(t *testing.T) {
	const key = "LOGFLUX_CONFIG_TEST_MISSING"

	t.Setenv(key, "")
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "config.yaml")

	if err := os.WriteFile(path, []byte("http:\n  address: ${"+key+"}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), key) {
		t.Fatal("expected an error naming the missing environment variable")
	}
}
