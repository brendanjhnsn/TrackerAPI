# Stage 1 — build the React frontend
FROM node:20-alpine AS frontend
WORKDIR /app
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2 — build the Go binary
FROM golang:1.22-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY API/     ./API/
COPY core/    ./core/
COPY modules/ ./modules/
RUN CGO_ENABLED=0 GOOS=linux go build -o tracker ./API/...

# Stage 3 — minimal final image
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=backend  /app/tracker  ./tracker
COPY --from=frontend /app/dist     ./dist
EXPOSE 8080
CMD ["./tracker"]
