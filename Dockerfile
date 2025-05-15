# ---- Build Stage ----
# Use an official Go image as the base image for building the application.
# Choose a Go version that matches your development environment.
FROM golang:1.24-alpine AS builder
WORKDIR /app

# Install necessary build tools for cgo and go-sqlite3
RUN apk add --no-cache gcc musl-dev sqlite-dev


# Copy go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy the source code into the container
COPY . .

# Build the Go application
# -o /app/main: Output the binary to /app/main inside the container
# CGO_ENABLED=0: Disable CGO to build a statically linked binary (important for small alpine images)
# -ldflags="-w -s": Strip debug information to reduce binary size (optional, but good for production)
RUN GOOS=linux go build -a -ldflags="-w -s" -o /app/main .

# ---- Runtime Stage ----
# Use a minimal base image for the runtime environment.
# Alpine Linux is a good choice for its small size.
FROM alpine:latest

# It's good practice to run as a non-root user for security
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

WORKDIR /app

# Copy the compiled binary from the builder stage
COPY --from=builder /app/main .

# Expose the port that your Gin application listens on (e.g., 8080)
EXPOSE 8080

# Command to run the application
# The binary is now at /app/main in this stage
ENTRYPOINT ["./main"]