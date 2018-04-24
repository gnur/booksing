#!/bin/bash

if [[ "${1}" == "deploy" ]]; then
    log "Building booksing"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/booksing -a -ldflags '-extldflags "-static"' .
    log "Stopping booksing on melebox"
    ssh deploy-melebox systemctl stop booksing
    log "Copying booksing to melebox"
    scp dist/booksing deploy-melebox:/usr/local/bin/booksing
    log "Starting booksing on melebox"
    ssh deploy-melebox systemctl start booksing
    log "Showing booksing status on melebox"
    ssh deploy-melebox systemctl status booksing
    exit 0
fi

if [[ "${1}" == "test" ]]; then
    go build -o booksing *.go

elif [[ "${1}" != "release" ]]; then
    go-bindata-assetfs static/...;
    go build -o booksing *.go
else

    for os in linux darwin windows; do
        for arch in 386 arm amd64; do
            echo "building $os $arch"
            CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -o ~/dist/booksing-$os-$arch -a -ldflags '-extldflags "-static"' .
        done
    done

fi
