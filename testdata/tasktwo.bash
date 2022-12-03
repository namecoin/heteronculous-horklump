#!/usr/bin/env bash

set -euo pipefail
shopt -s nullglob globstar

git clone -b starting_tasks "$CIRRUS_REPO_CLONE_URL"
cd heteronculous-horklump/Starting_Tasks/Task_Two


if timeout 10 go run main.go; then
    exit 0
else
    exit 1
fi