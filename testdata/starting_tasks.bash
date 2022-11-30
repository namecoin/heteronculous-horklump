#!/usr/bin/env bash

set -euo pipefail
shopt -s nullglob globstar

git clone -b starting_tasks https://github.com/namecoin/heteronculous-horklump && cd heteronculous-horklump
cd Starting_Tasks
cd "$TASK_NUMBER" && cd Testing_application
export GOBIN="$PWD" && go install hello.go
cd ..

go run main.go
