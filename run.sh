go fmt ./...
go test ./...

CGO_ENABLED=0 go build -o bin/server ./cmd/server
truncate -s 0 data/log/epox.log

echo 'start server'
bin/server server.scf
