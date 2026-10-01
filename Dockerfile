# -----------------------------------------------------------------------------------------------
#                             Stage 1: Builder 
# -----------------------------------------------------------------------------------------------

# Builder has all the required elements for selfexecuting (code, golang, C compiler, dependencies, exra packages etc.)
# "FROM:" = Use a prebuilt image that already has Go installed. We are not installing anything!
# "builder" = Docker's stage name
FROM golang:1.26-alpine AS builder

# "WORKDIR" = 1. Sets the current working directory
#             2. Creates /app if it doesn’t exist
#             3. All next commands run inside /app
WORKDIR /app

# Copy go.mod, go.sum if it exists (optional)
# Use go.mod for define the dependencies of our code
COPY go.mod go.sum ./

RUN go mod download

# apk is the package manager of Alpine. The "build-base" install the C compiler and the --no-cache doesn't store cache of the installed packages
RUN apk add --no-cache build-base

# Copying all source files into /app in container
COPY . .

# Build the binary (we can remove main.go at the end if we have ONLY the main.go in structure)
# Create the binary "forum" inside the container
RUN go build -o forum ./cmd/main.go


# -----------------------------------------------------------------------------------------------
#                             Stage 2: Image creation
# -----------------------------------------------------------------------------------------------

# Defining the alpine version
FROM alpine:latest

# Set working directory inside image
WORKDIR /app

# Copy the binary (forum) and all the neccessary folder the program needs from the builder stage
COPY --from=builder /app/forum .
COPY --from=builder /app/internal/db/migrations ./internal/db/migrations
COPY --from=builder /app/templates ./templates

# Exposing port 8080
# Just document (saying) that the container uses port 8080
EXPOSE 8080

# Automatically runs the binary when container starts
CMD ["./forum"]


# -----------------------------------------------------------------------------------------------
#                             Build and Run the image from bash
# -----------------------------------------------------------------------------------------------

# 1. Build the image from inside the project's folder. This is the "blueprint"
# docker build -t forum .

# 2. Create and Run the container
# Giving the container a specific name: docker run --name my-container -p 8080:8080 forum
# docker run -p 8080:8080 forum

# -----------------------------------------------------------------------------------------------
#                       Find, Terminate and Delete Images and Containers
# -----------------------------------------------------------------------------------------------

# Find Builds
    # docker buildx history ls

# Remove Build
    # docker buildx history rm <BUILD_ID>
    # OR
    # docker buildx history rm --all

# Find Containers
    # docker ps     | Shows running containers 
    # docker ps -a  | Shows ALL containers

# Terminate Container
    # docker stop <container_id_or_name>

# Remove Container
    # docker rm <container_id_or_name>

# Find Images
    # docker images | Shows ALL images

# Remove Image
    # docker rmi <image_name>