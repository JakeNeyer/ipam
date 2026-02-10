# Build web UI
FROM node:22-alpine AS web
WORKDIR /app/web
COPY web/package.json web/package-lock.json* ./
RUN npm ci
COPY web/ ./
RUN npm run build

# Build Go API
FROM golang:1.25-alpine AS go
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /ipam .

# Run
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=go /ipam .
COPY --from=web /app/web/dist ./static
ENV PORT=8080 STATIC_DIR=/app/static
EXPOSE 8080
CMD ["./ipam"]
