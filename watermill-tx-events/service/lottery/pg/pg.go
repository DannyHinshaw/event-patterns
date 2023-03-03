package pg

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type Config struct {
	Pass string `env:"DB_PASS" default:"password"`
	User string `env:"DB_USER" default:"watermill"`
	Host string `env:"DB_HOST" default:"localhost"`
	Name string `env:"DB_NAME" default:"watermill"`
	Port string `env:"DB_PORT" default:"5432"`
}

// PostgresDSN formats the postgres connection dsn from Config values.
func (c *Config) PostgresDSN(sslMode string) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.User, c.Pass, c.Host, c.Port, c.Name, sslMode)
}

// Connect handles acquiring a connection to postgres with retries configured by the backoff arg.
func Connect(dsn string, backoff *ExpBackOff) *sql.DB {
	attempt := backoff.Attempt()
	if attempt >= backoff.MaxRetries {
		log.Fatalln("failed to acquire postgres connection after max retries")
	}

	db, err := sql.Open("postgres", dsn)
	for err != nil {
		delay := backoff.Delay()
		log.Printf("failed connecting to postgres (attempt %d), retrying in %s seconds...", attempt, delay)

		time.Sleep(delay)
		return Connect(dsn, backoff)
	}

	err = db.Ping()
	if err != nil {
		log.Fatalf("error returned from db.Ping: %s", err)
	}

	return db
}
