# Use official Golang image as a builder
FROM golang:1.23.3 as builder

# Set environment variable for static binary
ENV CGO_ENABLED=0 GOOS=linux

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go app
RUN go build -o main .

# Use a minimal image to run the application
FROM alpine:latest

# Install certificates to enable HTTPS requests
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the compiled binary from the builder
COPY --from=builder /app/main .

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
