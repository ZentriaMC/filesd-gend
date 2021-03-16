#!/bin/sh
set -e

docker build \
	-t zentria/filesd-gend \
	-t docker.zentria.ee/svc/filesd-gend \
	-f docker/Dockerfile .
