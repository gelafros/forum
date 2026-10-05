#!/bin/bash
set -e

IMAGE_NAME="forum"
CONTAINER_NAME="forum-container"
PORT="8080"

print_usage() {
    echo "Usage: ./docker.sh [build | run | start | stop | restart | logs | clean]"
    echo ""
    echo "Commands:"
    echo "  (no args) / start  Build image and run container"
    echo "  build              Build the Docker image"
    echo "  run                Run the container (without rebuilding)"
    echo "  stop               Stop and remove the container"
    echo "  restart            Restart the container"
    echo "  logs               View live container logs"
    echo "  clean              Remove container and image"
}

build_image() {
    echo "==> Building Docker image '${IMAGE_NAME}'..."
    docker build -t "${IMAGE_NAME}" .
    echo "==> Image '${IMAGE_NAME}' built successfully."
}

stop_container() {
    # Stop any container publishing to our port
    local port_containers
    port_containers=$(docker ps --format '{{.ID}} {{.Ports}}' | grep -F ":${PORT}->" | awk '{print $1}' || true)
    if [ -n "$port_containers" ]; then
        echo "==> Stopping container(s) using port ${PORT}..."
        docker stop $port_containers >/dev/null 2>&1 || true
    fi

    # Remove forum-container if it exists
    if docker ps -a --format '{{.Names}}' | grep -Eq "^${CONTAINER_NAME}\$"; then
        echo "==> Removing existing container '${CONTAINER_NAME}'..."
        docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
    fi
    echo "==> Port ${PORT} and container '${CONTAINER_NAME}' are clear."
}

run_container() {
    stop_container
    echo "==> Starting container '${CONTAINER_NAME}' on http://localhost:${PORT}..."
    docker run -d --name "${CONTAINER_NAME}" -p "${PORT}:${PORT}" "${IMAGE_NAME}"
    echo "==> Forum is now live at: http://localhost:${PORT}"
}

case "$1" in
    build)
        build_image
        ;;
    run)
        run_container
        ;;
    stop)
        stop_container
        ;;
    restart)
        stop_container
        run_container
        ;;
    logs)
        docker logs -f "${CONTAINER_NAME}"
        ;;
    clean)
        stop_container
        echo "==> Removing Docker image '${IMAGE_NAME}'..."
        docker rmi "${IMAGE_NAME}" || true
        echo "==> Cleaned up successfully."
        ;;
    help|-h|--help)
        print_usage
        ;;
    start|"")
        build_image
        run_container
        ;;
    *)
        echo "Unknown command: $1"
        print_usage
        exit 1
        ;;
esac
