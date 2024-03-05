#!/bin/bash

function log {
    echo "> $(date +%T) $*"
}

log "generating static files"
cd web;
bun run generate -- --preset=static || exit 1
cd -

log "Building container and pushing"
skaffold build
