#!/bin/sh

docker run \
	--rm \
	-e LOG_LEVEL=INFO \
	-e RUN_LOCAL=true \
	-v "$PWD":/tmp/lint \
	ghcr.io/super-linter/super-linter:454ba4482ce2cd0c505bc592e83c06e1e37ade61
