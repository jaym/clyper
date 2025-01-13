# Use the official Golang image to create a build artifact.
FROM golang:1.23 as builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN CGO_ENABLED=1 GOOS=linux go build --tags fts5 -o clyper -a -ldflags '-linkmode external -extldflags "-static"' ./apps/clyper

# Start a new stage from scratch
FROM alpine:latest

# Install ffmpeg
RUN apk add --no-cache ffmpeg

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/clyper /clyper

EXPOSE 8991
# Command to run the executable
ENTRYPOINT ["/clyper", "serve"]
