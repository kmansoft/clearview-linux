#!/usr/bin/env bash

go run \
    ./server/server_main.go \
    -cvdir ./cv -a 0.0.0.0 -influx-db-uri http://localhost:8086
