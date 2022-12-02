#!/usr/bin/env bash

set -euo pipefail
shopt -s nullglob globstar

git clone -b starting_tasks "$CIRRUS_REPO_CLONE_URL"
cd heteronculous-horklump/Starting_Tasks/Task_One/Testing_application
export GOBIN="$PWD" && go install hello.go
cd ..

go run main.go
