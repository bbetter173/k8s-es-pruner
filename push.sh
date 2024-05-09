#!/bin/bash
docker buildx create --name amd64 --platform linux/amd64 --use
docker buildx build --platform linux/amd64,linux/arm64,linux/arm/v7 -t hewhowas/es-index-pruner:latest --push .
