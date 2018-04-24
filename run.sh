#!/bin/bash

export BOOK_DIR=./bookdump/
#export ALLOW_DELETES=true

./build.sh

echo "running now"
./booksing
