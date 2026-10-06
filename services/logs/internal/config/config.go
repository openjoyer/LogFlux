package config

import (
	"fmt"
	"os"
	"time"

	"go.yaml.in/yaml/v4"
)

type Config struct {
	Kafka           KafkaConfig      `yaml:"kafka"`
	Http            HttpConfig       `yaml:"http"`
	ClickHouse      ClickHouseConfig `yaml:"clickhouse"`
	Batch           BatchConfig      `yaml:"batch"`
	Writer          WriterConfig     `yaml:"writer"`
	ShutdownTimeout time.Duration    `yaml:"shutdown_timeout"`
}

type KafkaConfig struct {
	Brokers      []string      `yaml:"brokers"`
	Topic        string        `yaml:"topic"`
	ClientID     string        `yaml:"client_id"`
	BatchTimeout time.Duration `yaml:"batch_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	MinBytes     int           `yaml:"min_bytes"`
	MaxBytes     int           `yaml:"max_bytes"`
}

type HttpConfig struct {
	Address string `yaml:"address"`
	Port    string `yaml:"port"`
}

type ClickHouseConfig struct {
	TLS             bool          `yaml:"tls"`
	Addresses       []string      `yaml:"addresses"`
	Database        string        `yaml:"database"`
	Username        string        `yaml:"username"`
	Password        string        `yaml:"password"`
	DialTimeout     time.Duration `yaml:"dial_timeout"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

type BatchConfig struct {
	MaxEvents     int           `yaml:"max_events"`
	MaxBytes      int           `yaml:"max_bytes"`
	FlushInterval time.Duration `yaml:"flush_interval"`
}

type WriterConfig struct {
	InsertTimeout     time.Duration `yaml:"insert_timeout"`
	MaxAttempts       int           `yaml:"max_attempts"`
	RetryInitialDelay time.Duration `yaml:"retry_initial_delay"`
	RetryMaxDelay     time.Duration `yaml:"retry_max_delay"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var document yaml.Node
	if err := yaml.Unmarshal(data, &document); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if err := expandEnvironment(&document); err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := document.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}

	return cfg, nil
}

func expandEnvironment(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode && node.Tag == "!!str" {
		var missing string
		expanded := os.Expand(node.Value, func(key string) string {
			value, ok := os.LookupEnv(key)
			if !ok && missing == "" {
				missing = key
			}
			return value
		})
		if missing != "" {
			return fmt.Errorf("missing environment variable: %s", missing)
		}
		if expanded != node.Value {
			node.Value = expanded
			node.Tag = ""
		}
	}
	for _, child := range node.Content {
		if err := expandEnvironment(child); err != nil {
			return err
		}
	}
	return nil
}
