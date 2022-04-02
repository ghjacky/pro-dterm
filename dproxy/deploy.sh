#!/bin/bash
docker build -t dproxy .
docker run -itd -p 8001:8001 --name dproxy -v /var/run/docker.sock:/var/run/docker.sock --user root dproxy