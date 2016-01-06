#! /usr/bin/env bash

root=$(dirname $0)

go get -u github.com/geterns/logadpt && go build -o $root/bin/massive_down $root/src
