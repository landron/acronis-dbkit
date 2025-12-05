#!/bin/bash

# TODO: start a dockerized mariadb for testing

# -e: exit on error
# -u: undefined vars are errors
# -o pipefail: pipeline fails if any command fails
set -euo pipefail

start=$(date +%s)

# gvm use go1.20.14
GO=go
# GO=go1.20

$GO mod tidy

# gofumpt -l -w ./
# ~8 files
# gofumpt -l -w -extra ./

$GO build ./...

### dbrutil examples

$GO build -o .build ./dbrutil/examples/dbr-instrumentation-1
# MYSQL_USER=root MYSQL_DATABASE=dbkit_test ./.build/dbr-instrumentation-1
$GO build -o .build ./dbrutil/examples/dbr-instrumentation-2
# docker-compose -f testing/docker/docker-compose.yml up -d
# MYSQL_USER=root MYSQL_DATABASE=dbkit_test ./.build/dbr-instrumentation-2
# http://localhost:8080/metrics 
# http://localhost:8080/long_operation

### migrations examples

# MYSQL_USER=root MYSQL_DATABASE=dbkit_test go test -run Example
# MYSQL_DSN="root@tcp(localhost:3306)/dbkit_test" go test -run Example ./distrlock
#   MYSQL_DSN="root@tcp(localhost:3306)/dbkit_test" go test -run ExampleNewDBManager ./distrlock
export MYSQL_USER=root
export MYSQL_DATABASE=dbkit_test
export MYSQL_DSN="${MYSQL_USER}@tcp(localhost:3306)/${MYSQL_DATABASE}"

# github.com/acronis/go-dbkit/distrlock   108.021s
SLOW_PKGS=(
  "github.com/acronis/go-dbkit/distrlock"
  "github.com/acronis/go-dbkit/pgx"
  "github.com/acronis/go-dbkit/postgres"
)
FAST_PKGS=$(go list ./... | grep -v -F -e "${SLOW_PKGS[0]}" -e "${SLOW_PKGS[1]}" -e "${SLOW_PKGS[2]}")
echo "Running fast tests..."
go test -run Example
go test $FAST_PKGS
# run slow tests only if SLOW=1
if [ "${SLOW:-0}" = "1" ]; then
    echo "Running slow tests..."
    # go test -run Example ./distrlock # already run below
    go test "${SLOW_PKGS[@]}"
fi

echo
golangci-lint-v1 run -v --timeout 600s
# TODO: change configuration for certain folders
# golangci-lint-v1 run -c ./dbrutil/examples/.golangci.yml ./dbrutil/examples/...

if true; then
    echo
    golangci-lint run -v --timeout 600s --config .golangci.v2.yml
fi

echo
echo "All checks passed successfully at $(date +'%H:%M:%S %d.%m')"
end=$(date +%s)
elapsed=$((end - start))
echo "Elapsed time: ${elapsed} seconds"
