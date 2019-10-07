#!/bin/bash

workingdir=$(mktemp -d)

function log {
    echo "> $(date +%T) $*"
}

function cleanup {
    log "Killing booksing"
    kill $(jobs -p)
    log "Removing old workdir"
    rm -rf workingdir
}

trap 'cleanup' EXIT

log "building binary"
go build -o booksing ./cmd/server/

log "Creating temp workspace in ${workingdir}"
cp -a testdata $workingdir

export BOOKSING_LOGLEVEL=debug
export BOOKSING_DATABASE="firestore://booksing-erwin-land"
export GOOGLE_APPLICATION_CREDENTIALS="booksing-creds.json"
export BOOKSING_IMPORTDIR="${workingdir}/testdata/import"
export BOOKSING_BOOKDIR="${workingdir}/testdata/"


log "running booksing"
./booksing &

log "starting live yarn"
cd web
yarn serve


wait
log "cleaning up workspace"
rm -rf ${workdingdir}
