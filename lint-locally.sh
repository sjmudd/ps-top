#!/bin/sh

# Use a stable tag. The GitHub Action uses a different reference; the Docker image
# is typically tagged as 'latest'. Avoid explicit hashes.
IMAGE="ghcr.io/super-linter/super-linter:latest"

# Pull the image if not present locally
if ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
	echo "Pulling $IMAGE..."
	docker pull "$IMAGE"
fi

docker run \
	--rm \
	-e LOG_LEVEL=INFO \
	-e RUN_LOCAL=true \
	-e VALIDATE_GO=false \
	-e VALIDATE_GITHUB_ACTIONS_ZIZMOR=false \
	-e VALIDATE_JSCPD=false \
	-e VALIDATE_BIOME_FORMAT=false \
	-v "$PWD":/tmp/lint \
	"$IMAGE"
