set -o errexit  # Exit on most errors
set -o errtrace # Make sure any error trap is inherited
set -o nounset  # Disallow expansion of unset variables
set -o pipefail # Use last non-zero exit code in a pipeline
# set -o xtrace

go fmt ./...
go test ./...

CGO_ENABLED=0 go build -o bin/server ./cmd/server
truncate -s 0 data/log/epox.log

echo 'start server'
bin/server server.scf
