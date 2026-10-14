package config

import (
	"fmt"
	"os"
	"time"

	"go.yaml.in/yaml/v4"
)

type Config struct {
	HTTP            HTTPConfig     `yaml:"http"`
	Grpc            GrpcConfig     `yaml:"grpc"`
	Postgres        PostgresConfig `yaml:"postgres"`
	Auth            AuthConfig     `yaml:"auth"`
	ShutdownTimeout time.Duration  `yaml:"shutdown_timeout"`
}

type AuthConfig struct {
	SessionTTL time.Duration `yaml:"session_ttl"`
}

type HTTPConfig struct {
	Address string `yaml:"address"`
}

type GrpcConfig struct {
	Address   string              `yaml:"address"`
	Keepalive GrpcKeepaliveConfig `yaml:"keepalive"`
}

type GrpcKeepaliveConfig struct {
	MaxConnIdle     time.Duration `yaml:"max_conn_idle"`
	MaxConnAge      time.Duration `yaml:"max_conn_age"`
	MaxConnAgeGrace time.Duration `yaml:"max_conn_age_grace"`
	Time            time.Duration `yaml:"time"`
	Timeout         time.Duration `yaml:"timeout"`
	MinPingInterval time.Duration `yaml:"min_ping_interval"`
}

type PostgresConfig struct {
	Host            string        `yaml:"host"`
	Port            string        `yaml:"port"`
	Database        string        `yaml:"database"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
	ConnectTimeout  time.Duration `yaml:"connect_timeout"`
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
