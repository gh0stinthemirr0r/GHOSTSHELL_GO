# Stage 1: Build Stage
FROM golang:1.20 as builder

# Set the working directory inside the container
WORKDIR /app

# Copy the Go modules manifest and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ghostshell ./cmd/ghostshell/main.go

# Stage 2: Final Minimal Image
FROM alpine:latest

# Add certificates for HTTPS usage
RUN apk --no-cache add ca-certificates

# Set working directory inside the container
WORKDIR /root/

# Copy the compiled binary from the builder stage
COPY --from=builder /app/ghostshell .

# Command to run the application
CMD ["./ghostshell"]
