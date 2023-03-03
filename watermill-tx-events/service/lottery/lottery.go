package lottery

import (
	"context"
	"math/rand"
	"time"

	wm "github.com/ThreeDotsLabs/watermill"

	"github.com/dannyhinshaw/watermill-tx-events/service/lottery/store"
)

type lotteryStore interface {
	GetWinnerByID(ctx context.Context, id int) (*store.Winner, error)
	InsertWinner(ctx context.Context, lottery store.Winner) (*store.Winner, error)
}

type Service struct {
	config Config
	store  lotteryStore
	logger wm.LoggerAdapter
}

func NewService(config Config, s lotteryStore, logger wm.LoggerAdapter) *Service {
	logger = logger.With(wm.LogFields{"service": "lottery"})

	return &Service{
		config: config,
		logger: logger,
		store:  s,
	}
}

// Run picks a random user at fixed intervals and makes him win a lottery.
func (s *Service) Run(ctx context.Context) {
	lotteryID := 1
	users := []string{"Mike", "Dwight", "Jim", "Pamela"}
	for range time.Tick(time.Second * 5) {
		user := users[rand.Intn(len(users))]
		logger := s.logger.With(wm.LogFields{"user": user, "lottery_id": lotteryID})
		logger.Info("User has been picked as a winner", nil)

		winner := store.Winner{
			ID:   lotteryID,
			Name: user,
		}
		_, err := s.store.InsertWinner(ctx, winner)
		if err != nil {
			logger.Error("Handler failed", err, nil)
		}

		lotteryID++
	}
}
