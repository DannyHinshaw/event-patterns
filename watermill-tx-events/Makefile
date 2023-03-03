lint:
	golangci-lint run

lottery: export DB_HOST=localhost
lottery: export PUBSUB_EMULATOR_HOST=localhost:8085
lottery:
	go run cmd/lottery/main.go

prizer: export DB_HOST=localhost
prizer: export PUBSUB_EMULATOR_HOST=localhost:8085
prizer:
	go run cmd/prizer/main.go