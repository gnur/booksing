#!/bin/bash

ulimit -S -n 1024

workingdir=$(mktemp -d)
mkdir "${workingdir}/import"
mkdir "${workingdir}/db"

function log {
    echo "> $(date +%T) $*"
}

function cleanup {
    log "Killing background process"
    killall background
    log "Removing old workdir"
    rm -rf "$workingdir"
    log "Stopping docker container"
    docker stop meili
    log "Done. ✅"
}

trap 'cleanup' EXIT

log "Starting temp meilisearch docker container"
docker run --name "meili" -d --rm \
  -p 7700:7700 \
  getmeili/meilisearch:v1.6

log "Creating temp workspace in ${workingdir}"
cp -a testdata/import $workingdir/import/

export BOOKSING_LOGLEVEL=debug
export BOOKSING_ADMINUSER='unknown'
export BOOKSING_DATABASEDIR="${workingdir}/db"
export BOOKSING_IMPORTDIR="${workingdir}/import"
export BOOKSING_FAILDIR="${workingdir}/failed"
export BOOKSING_BOOKDIR="${workingdir}/"
export BOOKSING_SAVEINTERVAL="20s"
export BOOKSING_MQTTENABLED=true
export BOOKSING_MQTTHOST="tcp://sanny.aawa.nl:1883"
export BOOKSING_MQTTTOPIC="events"
export BOOKSING_MQTTCLIENTID="booksing"
export BOOKSING_BINDADDRESS="localhost:7133"
export BOOKSING_EVENTSPORT="localhost:8821"
export BOOKSING_ACCEPTEDLANGUAGES="nl,en"

cd web
bun run dev &
cd -


air
