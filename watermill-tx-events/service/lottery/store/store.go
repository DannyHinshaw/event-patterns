package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	wm "github.com/ThreeDotsLabs/watermill"
	wsql "github.com/ThreeDotsLabs/watermill-sql/pkg/sql"
	wfwd "github.com/ThreeDotsLabs/watermill/components/forwarder"
	wmsg "github.com/ThreeDotsLabs/watermill/message"

	"github.com/dannyhinshaw/watermill-tx-events/event"
	"github.com/dannyhinshaw/watermill-tx-events/service/lottery/pg"
)

type Statements struct {
	getWinnerByID *sql.Stmt
	insertWinner  *sql.Stmt
}

type Store struct {
	db          *sql.DB
	stmts       Statements
	logger      wm.LoggerAdapter
	fwdTopic    string
	pubSubTopic string
}

func New(db *sql.DB, logger wm.LoggerAdapter, fwdTopic, pubSubTopic string) *Store {
	logger = logger.With(wm.LogFields{"package": "store"})

	return &Store{
		db:          db,
		logger:      logger,
		fwdTopic:    fwdTopic,
		pubSubTopic: pubSubTopic,
	}
}

func (s *Store) Migrate(ctx context.Context) error {
	createWinnersTable, err := s.db.Prepare(
		`CREATE TABLE IF NOT EXISTS winners (
					id   INT NOT NULL PRIMARY KEY,
					name VARCHAR NOT NULL
	   			)`,
	)
	if err != nil {
		return fmt.Errorf("error creating createWinners sql statment: %w", err)
	}

	createLotteryFwdTable, err := s.db.Prepare(
		`CREATE TABLE IF NOT EXISTS watermill_lottery_forwarded_sql (
			"offset" SERIAL,
			uuid VARCHAR(36) NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			payload JSON DEFAULT NULL,
			metadata JSON DEFAULT NULL
		);`,
	)
	if err != nil {
		return fmt.Errorf("error creating createWinners sql statment: %w", err)
	}

	createLotteryOffsetsTable, err := s.db.Prepare(`
		CREATE TABLE IF NOT EXISTS watermill_offsets_lottery_forwarded_sql (
			consumer_group VARCHAR(255) NOT NULL,
			offset_acked BIGINT,
			offset_consumed BIGINT NOT NULL,
			PRIMARY KEY(consumer_group)
		);`,
	)
	if err != nil {
		return fmt.Errorf("error creating createWinners sql statment: %w", err)
	}

	err = pg.TxContext(ctx, s.db, nil, func(tx *sql.Tx) error {
		_, err = tx.StmtContext(ctx, createWinnersTable).ExecContext(ctx)
		if err != nil {
			return fmt.Errorf("error executing createWinnersTable statement: %w", err)
		}
		_, err = tx.StmtContext(ctx, createLotteryFwdTable).ExecContext(ctx)
		if err != nil {
			return fmt.Errorf("error executing createLotteryFwdTable statement: %w", err)
		}
		_, err = tx.StmtContext(ctx, createLotteryOffsetsTable).ExecContext(ctx)
		if err != nil {
			return fmt.Errorf("error executing createLotteryOffsetsTable statement: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("error creating winners table: %w", err)
	}

	if err = s.PrepareStatements(); err != nil {
		return fmt.Errorf("error preparing store sql statements: %w", err)
	}

	return nil
}

func (s *Store) PrepareStatements() error {
	getWinnerByID, err := s.db.Prepare(
		`SELECT * FROM winners WHERE id = $1`,
	)
	if err != nil {
		return fmt.Errorf("error creating getWinnerByID sql statment: %w", err)
	}

	insertWinner, err := s.db.Prepare(
		`INSERT INTO winners (id, name) VALUES ($1, $2) RETURNING id`,
	)
	if err != nil {
		return fmt.Errorf("error creating getWinnerByID sql statment: %w", err)
	}

	s.stmts = Statements{
		getWinnerByID: getWinnerByID,
		insertWinner:  insertWinner,
	}

	return nil
}

func (s *Store) GetWinnerByID(ctx context.Context, id int) (*Winner, error) {
	var row Winner
	err := pg.TxContext(ctx, s.db, nil, func(tx *sql.Tx) error {
		_, err := tx.StmtContext(ctx, s.stmts.getWinnerByID).ExecContext(ctx, id)
		if err != nil {
			return fmt.Errorf("error executing getWinnerByID statement: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error getting winner with id %d: %w", id, err)
	}

	return &row, nil
}

func (s *Store) InsertWinner(ctx context.Context, winner Winner) (*Winner, error) {
	evt := event.LotteryConcluded{WinnerID: winner.ID}
	payload, err := json.Marshal(evt)
	if err != nil {
		return nil, fmt.Errorf("publishWinner: error marshalling json for lottery concluded: %w", err)
	}

	err = pg.TxContext(ctx, s.db, nil, func(tx *sql.Tx) error {
		var obp *wfwd.Publisher
		obp, err = s.txPublisher(tx)
		if err != nil {
			return fmt.Errorf("store.InsertWinner: error creating outboxPublisher: %w", err)
		}

		_, err = tx.StmtContext(ctx, s.stmts.insertWinner).ExecContext(ctx, winner.ID, winner.Name)
		if err != nil {
			return fmt.Errorf("store.InsertWinner: error executing insertWinner statement: %w", err)
		}

		err = obp.Publish(s.pubSubTopic, wmsg.NewMessage(wm.NewULID(), payload))
		if err != nil {
			return fmt.Errorf("store.InsertWinner: error publishing lottery concluded: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("store.InsertWinner: error inserting winner (%+v): %w", winner, err)
	}

	return &winner, nil
}

// txPublisher creates a new transaction scoped sql publisher wrapped in a forwarder.
func (s *Store) txPublisher(tx wsql.ContextExecutor) (*wfwd.Publisher, error) {
	sqlPubConfig := wsql.PublisherConfig{
		SchemaAdapter: wsql.DefaultPostgreSQLSchema{},
	}

	publisher, err := wsql.NewPublisher(tx, sqlPubConfig, s.logger)
	if err != nil {
		return nil, fmt.Errorf("store.txPublisher: error creating new sql publisher: %s", err)
	}

	return wfwd.NewPublisher(publisher, wfwd.PublisherConfig{
		ForwarderTopic: s.fwdTopic,
	}), nil
}
