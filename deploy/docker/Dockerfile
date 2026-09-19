# Stage 1: Build Frontend (SPA)
FROM node:20-alpine AS frontend-builder
WORKDIR /app
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Backend (Single Binary)
FROM golang:1.24-alpine AS backend-builder
WORKDIR /app
# Pre-download dependencies
COPY backend/go.mod backend/go.sum ./
RUN go mod download
# Copy backend source
COPY backend/ ./
# Copy built frontend for embedding (Go:embed build/*)
COPY --from=frontend-builder /app/build ./cmd/gitpatrol/build
# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build -o gitpatrol ./cmd/gitpatrol/main.go

# Stage 3: Final Minimal Runtime
FROM alpine:latest
# Install Git (required for mirroring) and Curl (for health checks)
RUN apk add --no-cache git curl ca-certificates
WORKDIR /app
# Copy single binary from builder
COPY --from=backend-builder /app/gitpatrol .
# Volume for persistent data (mirrors, database)
VOLUME /var/lib/gitpatrol
# Default Environment Variables
ENV DB_PATH=/var/lib/gitpatrol/db/gitpatrol.db
ENV DATA_DIR=/var/lib/gitpatrol/data
ENV WORKERS=3
ENV PORT=8080
EXPOSE 8080
# Start GitPatrol
CMD ["./gitpatrol"]
