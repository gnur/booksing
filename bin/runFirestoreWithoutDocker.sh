#!/bin/bash

workingdir=$(mktemp -d)
mkdir "${workingdir}/import"

if [[ -f ~/.global-env ]]; then
    source ~/.global-env
fi

function log {
    echo "> $(date +%T) $*"
}

function cleanup {
    log "Killing booksing"
    kill $(jobs -p)
    log "Removing old workdir"
    rm -rf workingdir
    log "Emptying database"
    firebase firestore:delete -y -r /envs/dev
    log "Done. ✅"
}

trap 'cleanup' EXIT

#log "building static assets"
#cd web; yarn build; cd -;
#go-bindata web/dist/...; mv bindata.go cmd/server/bindata.go

log "building binary"
go build -o booksing ./cmd/server/ || exit 1

log "Creating temp workspace in ${workingdir}"
cp -a testdata/import/gutenberg/* $workingdir/import/

export BOOKSING_LOGLEVEL=debug
export BOOKSING_DATABASE="firestore://booksing"
export GOOGLE_APPLICATION_CREDENTIALS="booksing-creds.json"
export BOOKSING_IMPORTDIR="${workingdir}/import"
export BOOKSING_BOOKDIR="${workingdir}/"


log "running booksing"
./booksing &

log "starting live yarn"
cd web
yarn serve


wait
