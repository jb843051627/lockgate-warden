#!/usr/bin/env bash
# 用法: ./build_benzhi_docker.sh <名称> <平台>
# 例:   ./build_benzhi_docker.sh lockgate-warden linux/amd64
set -euo pipefail

NAME="${1:?usage: build_benzhi_docker.sh <name> <platform>}"
PLATFORM="${2:?usage: build_benzhi_docker.sh <name> <platform>}"

docker buildx build \
  --platform "$PLATFORM" \
  -f benzhi.Dockerfile \
  -t "benzhi/${NAME}:latest" \
  --load \
  .
