# Stage 1: Build the Go server
FROM golang:1.18-alpine AS builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go build -o main .

# Stage 2: Run the Go server
FROM alpine:latest

# Copy the pre-built binary from the builder stage
COPY --from=builder /app/main /usr/local/bin/main

# Expose the port the app runs on
EXPOSE 8080

# Start the Go server
CMD ["/usr/local/bin/main", "8080"]