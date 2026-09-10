package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Redis    RedisConfig    `json:"redis"`
	Raft     RaftConfig     `json:"raft"`
	AI       AIConfig       `json:"ai"`
	Log      LogConfig      `json:"log"`
}

type ServerConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

func (s ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

type DatabaseConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	User         string        `json:"user"`
	Password     string        `json:"password"`
	Name         string        `json:"name"`
	SSLMode      string        `json:"ssl_mode"`
	MaxOpenConns int           `json:"max_open_conns"`
	MaxIdleConns int           `json:"max_idle_conns"`
	MaxLifetime  time.Duration `json:"max_lifetime"`
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.Name, d.SSLMode,
	)
}

type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

type RaftConfig struct {
	NodeID          string        `json:"node_id"`
	Peers           []string      `json:"peers"`
	ElectionTimeout time.Duration `json:"election_timeout"`
	HeartbeatTick   int           `json:"heartbeat_tick"`
	DataDir         string        `json:"data_dir"`
}

type AIConfig struct {
	OllamaURL  string `json:"ollama_url"`
	Model      string `json:"model"`
	Enabled    bool   `json:"enabled"`
	MaxTokens  int    `json:"max_tokens"`
}

type LogConfig struct {
	Level  string `json:"level"`
	Format string `json:"format"`
	Output string `json:"output"`
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Database: DatabaseConfig{
			Host:         "localhost",
			Port:         5432,
			User:         "atlas",
			Password:     "atlas",
			Name:         "atlas",
			SSLMode:      "disable",
			MaxOpenConns: 25,
			MaxIdleConns: 5,
			MaxLifetime:  5 * time.Minute,
		},
		Redis: RedisConfig{
			Addr: "localhost:6379",
			DB:   0,
		},
		Raft: RaftConfig{
			NodeID:          "node-1",
			ElectionTimeout: 1500 * time.Millisecond,
			HeartbeatTick:   1,
			DataDir:         "./data/raft",
		},
		AI: AIConfig{
			OllamaURL: "http://localhost:11434",
			Model:     "llama3.2",
			Enabled:   false,
			MaxTokens: 2048,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
		},
	}
}

func Load(path string) (*Config, error) {
	cfg := Default()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config file: %w", err)
		}
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config file: %w", err)
		}
	}

	cfg.applyEnvOverrides()
	return cfg, nil
}

func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("ATLAS_HOST"); v != "" {
		c.Server.Host = v
	}
	if v := os.Getenv("ATLAS_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Server.Port = port
		}
	}
	if v := os.Getenv("ATLAS_DB_HOST"); v != "" {
		c.Database.Host = v
	}
	if v := os.Getenv("ATLAS_DB_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			c.Database.Port = port
		}
	}
	if v := os.Getenv("ATLAS_DB_USER"); v != "" {
		c.Database.User = v
	}
	if v := os.Getenv("ATLAS_DB_PASSWORD"); v != "" {
		c.Database.Password = v
	}
	if v := os.Getenv("ATLAS_DB_NAME"); v != "" {
		c.Database.Name = v
	}
	if v := os.Getenv("ATLAS_REDIS_ADDR"); v != "" {
		c.Redis.Addr = v
	}
	if v := os.Getenv("ATLAS_LOG_LEVEL"); v != "" {
		c.Log.Level = v
	}
	if v := os.Getenv("ATLAS_RAFT_NODE_ID"); v != "" {
		c.Raft.NodeID = v
	}
	if v := os.Getenv("ATLAS_AI_ENABLED"); v == "true" {
		c.AI.Enabled = true
	}
}
