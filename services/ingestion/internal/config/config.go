package config

import (
	"fmt"
	"os"
	"time"

	"go.yaml.in/yaml/v4"
)

type Config struct {
	HTTP     HTTPConfig     `yaml:"http"`
	Kafka    KafkaConfig    `yaml:"kafka"`
	Redis    RedisConfig    `yaml:"redis"`
	Postgres PostgresConfig `yaml:"postgres"`
}

type KafkaConfig struct {
	Brokers      []string      `yaml:"brokers"`
	Topic        string        `yaml:"topic"`
	ClientID     string        `yaml:"client_id"`
	BatchTimeout time.Duration `yaml:"batch_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

type RedisConfig struct {
	Address      string        `yaml:"address"`
	DB           int           `yaml:"db"`
	DialTimeout  time.Duration `yaml:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	Password     string        `yaml:"password"`
}

type HTTPConfig struct {
	Address string `yaml:"address"`
	Port    string `yaml:"port"`
}

type PostgresConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
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
