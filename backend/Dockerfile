# Multi-stage build for Go backend
# Needs full workspace context because backend imports internal/ and scraper/

FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy workspace files
COPY go.work go.work
COPY backend/go.mod backend/go.sum backend/
COPY internal/go.mod internal/go.sum internal/
COPY scraper/go.mod scraper/go.sum scraper/

# Download dependencies for all modules
RUN cd backend && go mod download
RUN cd internal && go mod download
RUN cd scraper && go mod download

# Copy source code
COPY backend/ backend/
COPY internal/ internal/
COPY scraper/ scraper/

# Build the backend binary
RUN cd backend && CGO_ENABLED=0 GOOS=linux go build -o /app/server .

# Runtime stage
FROM alpine:3.21

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]
