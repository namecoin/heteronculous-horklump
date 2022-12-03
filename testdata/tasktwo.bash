#!/usr/bin/env bash

set -euo pipefail
shopt -s nullglob globstar

Check() {
    if grep -i "socket" dunp.txt; then
        exit 0
    else
        exit 1
    fi
}

git clone -b starting_tasks "$CIRRUS_REPO_CLONE_URL"
cd heteronculous-horklump/Starting_Tasks/Task_Two

timeout 10 go run main.go >> dump.txt ; Check



