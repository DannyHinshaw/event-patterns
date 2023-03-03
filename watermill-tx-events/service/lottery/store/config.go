package store

import (
	"github.com/dannyhinshaw/watermill-tx-events/service/lottery/pg"
)

type DBConfig struct {
	pg.Config
	ForwarderSQLTopic string `env:"FORWARDER_SQL_TOPIC"`
}
