#!/bin/sh

docker run -e RUN_LOCAL=true -v "$PWD":/tmp/lint --rm ghcr.io/super-linter/super-linter:v8.5.0
