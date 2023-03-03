package main

import (
	"context"
	"log"
	"time"

	"github.com/dannyhinshaw/watermill-tx-events/service/prizer"

	"github.com/ThreeDotsLabs/watermill"
	_ "github.com/lib/pq"
)

var (
	logger = watermill.NewStdLogger(false, false)
)

func main() {
	conf, err := prizer.GetConfig()
	if err != nil || conf == nil {
		log.Fatal("error getting prizer config.")
	}
	config := *conf

	ps := prizer.NewService(config.GCPConfig, logger)
	go func() {
		err = ps.Run(context.Background())
		if err != nil {
			log.Fatalf("sender.main: error running prize sender service: %s", err)
		}
	}()

	time.Sleep(time.Second * 60)
}
