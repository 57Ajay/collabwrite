# Dockerfile

# --- Stage 1: The Builder ---
# We use a full Go image to build our application binary.
FROM golang:1.25-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy the go.mod and go.sum files to download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the application, creating a static binary.
# CGO_ENABLED=0 is important for creating a static binary that can run in a minimal image.
RUN CGO_ENABLED=0 GOOS=linux go build -o /collab-backend .

# --- Stage 2: The Final Image ---
# We use a minimal alpine image which is very small and secure.
FROM alpine:latest

# Set the working directory
WORKDIR /

# Copy ONLY the compiled binary from the 'builder' stage.
COPY --from=builder /collab-backend /collab-backend

# This command will be run when the container starts.
ENTRYPOINT ["/collab-backend"]
