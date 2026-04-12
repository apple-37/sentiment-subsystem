package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all subsystem configuration loaded from config.yaml.
type Config struct {
	MongoDB   MongoDBConfig   `yaml:"mongodb"`
	Output    OutputConfig    `yaml:"output"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
}

type MongoDBConfig struct {
	URI        string `yaml:"uri"`
	Database   string `yaml:"database"`
	Collection string `yaml:"collection"`
	// Timeout for a single MongoDB operation.
	TimeoutSeconds int `yaml:"timeout_seconds"`
}

type OutputConfig struct {
	// FilePath is the absolute or relative path where the .txt (JSONL) file is written.
	FilePath string `yaml:"file_path"`
	// TempSuffix is appended while writing; the file is atomically renamed on success.
	TempSuffix string `yaml:"temp_suffix"`
}

type SchedulerConfig struct {
	// Interval between export runs, e.g. "30s", "5m", "1h".
	Interval time.Duration `yaml:"interval"`
}

// Load reads and validates the YAML config at path.
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config file: %w", err)
	}
	defer f.Close()

	var cfg Config
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Apply defaults.
	if cfg.MongoDB.TimeoutSeconds == 0 {
		cfg.MongoDB.TimeoutSeconds = 30
	}
	if cfg.Output.TempSuffix == "" {
		cfg.Output.TempSuffix = ".tmp"
	}
	if cfg.Scheduler.Interval == 0 {
		cfg.Scheduler.Interval = 5 * time.Minute
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if c.MongoDB.URI == "" {
		return fmt.Errorf("mongodb.uri is required")
	}
	if c.MongoDB.Database == "" {
		return fmt.Errorf("mongodb.database is required")
	}
	if c.MongoDB.Collection == "" {
		return fmt.Errorf("mongodb.collection is required")
	}
	if c.Output.FilePath == "" {
		return fmt.Errorf("output.file_path is required")
	}
	return nil
}
