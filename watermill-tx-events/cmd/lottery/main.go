package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/dannyhinshaw/watermill-tx-events/service/lottery"
	"github.com/dannyhinshaw/watermill-tx-events/service/lottery/pg"
	"github.com/dannyhinshaw/watermill-tx-events/service/lottery/store"

	wm "github.com/ThreeDotsLabs/watermill"
	wgcp "github.com/ThreeDotsLabs/watermill-googlecloud/pkg/googlecloud"
	wsql "github.com/ThreeDotsLabs/watermill-sql/pkg/sql"
	wfwd "github.com/ThreeDotsLabs/watermill/components/forwarder"
)

var (
	logger = wm.NewStdLogger(false, false)
)

func main() {
	conf, err := lottery.GetConfig()
	if err != nil || conf == nil {
		log.Fatal("error getting prizer config.")
	}
	config := *conf
	dbConf := config.DBConfig

	psqlConn := dbConf.PostgresDSN("disable")
	db := pg.Connect(psqlConn, &pg.ExpBackOff{
		Coefficient: 2,
		MaxDelay:    10,
		MaxRetries:  10,
	})

	s := store.New(db, logger, dbConf.ForwarderSQLTopic, config.EventTopic)
	if err = s.Migrate(context.Background()); err != nil {
		log.Fatalln("lottery.main: error migrating store", err)
	}

	outboxSubscriber, err := newDatabasePubSubPipe(db, logger, config.ProjectID, config.ForwarderSQLTopic)
	if err != nil {
		log.Fatalf("lottery.main: error creating outbox sql subscription forwarder: %s", err)
	}

	go func() {
		defer outboxSubscriber.Close()
		err = outboxSubscriber.Run(context.Background())
		if err != nil {
			log.Fatalf("lottery.main: error running forwarder: %s", err)
		}
	}()

	lotto := lottery.NewService(config, s, logger)
	go lotto.Run(context.Background())

	time.Sleep(time.Second * 60)
}

// newDatabasePubSubPipe creates a a subscription to a sql table of events and pipes them to GCP PubSub.
func newDatabasePubSubPipe(db *sql.DB, logger wm.LoggerAdapter, projectID, sqlFwdTopic string) (*wfwd.Forwarder, error) {
	// Custom adapters do *not* have SchemaInitializer methods because table
	// creation should be handled by SQL migrations on a separate level,
	// so InitializeSchema must be "false", or else it will error.
	sqlSubConfig := wsql.SubscriberConfig{
		SchemaAdapter:  wsql.DefaultPostgreSQLSchema{},
		OffsetsAdapter: wsql.DefaultPostgreSQLOffsetsAdapter{},
	}

	sqlSubscriber, err := wsql.NewSubscriber(db, sqlSubConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("error creating sqlSubscriber: %w", err)
	}

	gcpConfig := wgcp.PublisherConfig{
		ProjectID: projectID,
	}

	gcpPublisher, err := wgcp.NewPublisher(gcpConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("error creating gcpPublisher: %w", err)
	}

	return wfwd.NewForwarder(sqlSubscriber, gcpPublisher, logger, wfwd.Config{
		ForwarderTopic: sqlFwdTopic,
	})
}
