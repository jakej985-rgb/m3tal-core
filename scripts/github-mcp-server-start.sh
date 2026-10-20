#!/bin/bash
# Reuse existing github-mcp-server container if already running,
# otherwise start a new one. Prevents duplicate MCP containers.

CONTAINER_NAME="github-mcp-server"

if docker ps --format "{{.Names}}" | grep -q "^${CONTAINER_NAME}$"; then
  # Container already running — exec into it to share the instance
  exec docker exec -i "$CONTAINER_NAME" /server/github-mcp-server stdio
else
  # No running container — start a fresh named one
  exec docker run -i --rm \
    --name "$CONTAINER_NAME" \
    -e GITHUB_PERSONAL_ACCESS_TOKEN \
    ghcr.io/github/github-mcp-server
fi
