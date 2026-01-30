#!/bin/bash

wait_for_database() {
    # git checkout dev -- testing/docker/docker-compose.yml check.sh

    echo "Waiting for database to be ready..."
    timeout 60 bash -c 'until docker compose -f testing/docker/docker-compose.yml ps mysql | grep -q "(healthy)"; do sleep 2; done'

    # Test database connectivity
    echo "Testing database connectivity..."
    timeout 30 bash -c 'until docker exec mysql mariadb -uroot -e "SELECT 1" >/dev/null 2>&1; do
      echo "Database not ready yet, waiting...";
      sleep 2;
    done'

    # # Test schema is ready
    # echo "Waiting for schema to be ready..."
    # timeout 60 bash -c 'until docker exec mysql mysql -uroot -e "USE snactivation; SELECT 1" >/dev/null 2>&1; do
    #   echo "Schema not ready yet, waiting...";
    #   sleep 1;
    # done'

    echo "Database is ready, running tests..."
}

# -e: exit on error
# -u: undefined vars are errors
# -o pipefail: pipeline fails if any command fails
set -euo pipefail

start=$(date +%s)

# gvm use go1.24.12
GO=go
# GO=go1.24

$GO mod tidy

# gofumpt -l -w ./
# ~8 files
# gofumpt -l -w -extra ./

$GO build ./...

### dbrutil examples

$GO build -o .build ./dbrutil/examples/dbr-instrumentation-1
# MYSQL_USER=root MYSQL_DATABASE=dbkit_test ./.build/dbr-instrumentation-1
$GO build -o .build ./dbrutil/examples/dbr-instrumentation-2

docker compose -f testing/docker/docker-compose.yml up -d
wait_for_database

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

docker compose -f testing/docker/docker-compose.yml down

echo
golangci-lint run -v --timeout 600s

echo
echo "All checks passed successfully at $(date +'%H:%M:%S %d.%m')"
end=$(date +%s)
elapsed=$((end - start))
echo "Elapsed time: ${elapsed} seconds"
