#!/bin/bash

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

$GO build -o .build ./dbrutil/examples/dbr-instrumentation-1
# MYSQL_USER=root MYSQL_DATABASE=dbkit_test ./.build/dbr-instrumentation-1
$GO build -o .build ./dbrutil/examples/dbr-instrumentation-2
# MYSQL_USER=root MYSQL_DATABASE=dbkit_test ./.build/dbr-instrumentation-2
# http://localhost:8080/metrics 
# http://localhost:8080/long_operation

# github.com/acronis/go-dbkit/distrlock   108.021s
SLOW_PKGS=(
  "github.com/acronis/go-dbkit/distrlock"
  "github.com/acronis/go-dbkit/pgx"
  "github.com/acronis/go-dbkit/postgres"
)
FAST_PKGS=$(go list ./... | grep -v -F -e "${SLOW_PKGS[0]}" -e "${SLOW_PKGS[1]}" -e "${SLOW_PKGS[2]}")
echo "Running fast tests..."
go test $FAST_PKGS
# run slow tests only if SLOW=1
if [ "${SLOW:-0}" = "1" ]; then
    echo "Running slow tests..."
    go test "${SLOW_PKGS[@]}"
fi

echo
golangci-lint-v1 run -v --timeout 600s
# TODO: it does not work
# golangci-lint-v1 run -c ./dbrutil/examples/.golangci.yml ./dbrutil/examples/...

if false; then
    # 49 issues:
    # * contextcheck: 2
    # * errcheck: 2
    # * errorlint: 1
    # * goconst: 2
    # * gofumpt: 3
    # * gosec: 1
    # * lll: 1
    # * nilnil: 1
    # * nlreturn: 6
    # * noctx: 5
    # * nolintlint: 16
    # * prealloc: 2
    # * rowserrcheck: 3
    # * staticcheck: 1
    # * thelper: 3
    echo
    golangci-lint run -v --timeout 600s --config .golangci.v2.yml
fi

echo
echo "All checks passed successfully at $(date +'%H:%M:%S %d.%m')"
end=$(date +%s)
elapsed=$((end - start))
echo "Elapsed time: ${elapsed} seconds"
