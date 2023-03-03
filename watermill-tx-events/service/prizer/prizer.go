package prizer

import (
	"context"
	"encoding/json"
	"fmt"

	wm "github.com/ThreeDotsLabs/watermill"
	wmgcp "github.com/ThreeDotsLabs/watermill-googlecloud/pkg/googlecloud"

	"github.com/dannyhinshaw/watermill-tx-events/event"
)

type Service struct {
	config GCPConfig
	logger wm.LoggerAdapter
}

func NewService(config GCPConfig, logger wm.LoggerAdapter) *Service {
	logger = logger.With(wm.LogFields{"service": "prize_sender"})

	return &Service{
		config: config,
		logger: logger,
	}
}

// Run listens to UserWonLottery events and sends a prize straight to the user that has won.
func (s *Service) Run(ctx context.Context) error {
	gcpSubConf := wmgcp.SubscriberConfig{
		ProjectID: s.config.ProjectID,
	}

	gcpSub, err := wmgcp.NewSubscriber(gcpSubConf, s.logger)
	if err != nil {
		return fmt.Errorf("error getting new gcp subscriber for project %s: %w", s.config.ProjectID, err)
	}

	topic := s.config.EventTopic
	events, err := gcpSub.Subscribe(ctx, topic)
	if err != nil {
		s.logger.Error("error subscribing to gcp pubsub topic:", err, wm.LogFields{"topic": topic})
		return err
	}

	s.logger.Info("successfully subscribed to gcp pubsub topic", wm.LogFields{"topic": topic})
	for rawEvent := range events {
		evt := event.LotteryConcluded{}
		err = json.Unmarshal(rawEvent.Payload, &evt)
		if err != nil {
			return fmt.Errorf("error json unmarshalling payload %s: %w", rawEvent.Payload, err)
		}

		rawEvent.Ack()

		s.logger.Info("Sending a prize to the winner", wm.LogFields{
			"winnerID": evt.WinnerID,
		})
	}

	return nil
}
