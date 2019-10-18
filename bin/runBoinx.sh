#!/bin/bash

function log {
    echo "> $(date +%T) $*"
}

if [[ -f ~/.global-env ]]; then
    source ~/.global-env
fi


log "building binary"
go build -o boinx ./cmd/boinx/ || exit 1


./boinx \
    -booksing-host "$BOOKSING_HOST" \
    -bucket "booksing" \
    -host "$S3_HOST" \
    -import-dir "testdata/import" \
    -api-key "$BOOKSING_API_KEY"
