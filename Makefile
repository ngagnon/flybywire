default:

fly-server: bin/fly-server

bin/fly-server:
	go build -o bin/fly-server ./cmd/fly-server

test: fly-server
	go test ./internal/db
	go test ./internal/vfs
	go test ./internal/crypto
	go test ./internal/wire
	bundle exec rspec