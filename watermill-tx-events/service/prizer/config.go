package prizer

import (
	"fmt"
	"github.com/jinzhu/configor"

	"github.com/dannyhinshaw/watermill-tx-events/service/lottery/pg"
)

type GCPConfig struct {
	EventTopic string `env:"GCP_PUBSUB_EVENT_TOPIC"`
	ProjectID  string `env:"GCP_PROJECT_ID"`
}

type Config struct {
	GCPConfig
	DBConfig pg.Config
}

// GetConfig handles processing the environment variables to configure the script run
func GetConfig() (*Config, error) {
	var c Config
	cfg := configor.New(&configor.Config{})
	if err := cfg.Load(&c); err != nil {
		return nil, fmt.Errorf("prizer.GetConfig: encountered error loading config from env: %w", err)
	}

	return &c, nil
}
