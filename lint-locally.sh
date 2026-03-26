#!/bin/sh

docker run \
	--rm \
	-e LOG_LEVEL=INFO \
	-e RUN_LOCAL=true \
	-v "$PWD":/tmp/lint \
	ghcr.io/super-linter/super-linter:latest
