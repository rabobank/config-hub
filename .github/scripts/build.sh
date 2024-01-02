#!/bin/bash

OUTPUT_DIR=$PWD/dist
mkdir -p ${OUTPUT_DIR}

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $PWD/target/linux_amr64/config-hub -ldflags "-X github.com/rabobank/config-hub/cfg.Version=${VERSION} -X github.com/rabobank/config-hub/cfg.Commit=${COMMIT}"
tar czf ${OUTPUT_DIR}/config-hub-linux-amd64-${VERSION}.tgz -C $PWD/target/linux_amr64 config-hub
