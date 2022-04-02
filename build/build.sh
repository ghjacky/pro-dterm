#!/bin/bash

rm dterm dterm.tgz 2> /dev/null

cd ..

GOOS=linux GOARCH=amd64 go build -o dterm


tar zcvf dterm.tgz dterm configs

mv dterm dterm.tgz build 

cd build

docker build -t dev-myapp.local/dterm .
