#!/bin/sh
set -e

docker build \
	-t zentria/prometheus-file-gen \
	-t docker.zentria.ee/svc/prometheus-file-gen \
	-f docker/Dockerfile .
