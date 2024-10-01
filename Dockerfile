# Install dependencies
FROM golang:1.22.7-bookworm AS deps

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

# Build
FROM golang:1.22.7-bookworm AS build

WORKDIR /app

COPY --from=deps /go/pkg /go/pkg
COPY . .

# Enable them if you need them
# ENV CGO_ENABLED=0
# ENV GOOS=linux

RUN go build -ldflags="-w -s" -o ./bin/main ./main.go

# Run
FROM debian:bookworm-slim

WORKDIR /app

# Create a non-root user and group
RUN groupadd -r appuser && useradd -r -g appuser appuser

# Copy the built application
COPY --from=build /app/bin/main .

# Change ownership of the application binary
RUN chown appuser:appuser /app/main

# Switch to the non-root user
USER appuser

CMD ["./main"]
