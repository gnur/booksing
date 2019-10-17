#!/bin/bash

version=""
full=false
cleanup=false
dryrun=false

while getopts "v:fcn" opt; do
    case $opt in
      v)
        version="$OPTARG"
        ;;
      f)
        full=true
        ;;
      c)
        cleanup=true
        ;;
      n)
        dryrun=true
        ;;
      \?)
        echo "Invalid option: -$OPTARG"
        echo
        help 1
        ;;
      :)
        echo "Option -$OPTARG requires an argument"
        echo
        help 1
        ;;
    esac
done


function log {
    echo "> $(date +%T) $*"
}



if $full; then
    log "Generating static assets"
    cd web || exit
    yarn
    yarn build
    cd - || exit

    log "binpacking static assets"
    go-bindata web/dist/...; mv bindata.go cmd/server/bindata.go
fi

log "adding version to app.yaml"
if [[ "${version}" == "" ]]; then
    git fetch --tags
    version=$(git describe --tags)
    if [[ $? -ne 0 ]]; then
        version=$(git rev-parse --short HEAD)
        if [[ $? -ne 0 ]]; then
          echo "Please set the version or use this in a initialized git repository"
          echo
          help 1
        fi
    fi
    if [[ $(git status --short | wc -l) -gt 0 ]]; then
        version="${version}-$(date +%F-%T)-dirty"
    fi
fi
log "Current version: ${version}"

cd cmd/server || exit 1
sed -i '.bak' 's/<<REPLACE_ME>>/'"${version}"'/g'  env_vars.yaml

success=true
if $dryrun; then
    log "not deploying because of -n"
else
    if ! gcloud app deploy app.yaml -q --project booksing; then
        log "Deployment failed, check app.yaml"
        success=false
    fi
fi

mv env_vars.yaml.bak env_vars.yaml

if $success; then
  log "Deployed app in ${SECONDS} seconds"
  alert "Deployed cronvict in ${SECONDS} seconds"
fi
