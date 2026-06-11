# Stage 1: Build
FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o app ./cmd/server

# Stage 2: Final (ultra-lightweight)
FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /build/app .
COPY --from=builder /build/templates ./templates
COPY --from=builder /build/static ./static
RUN mkdir -p /app/data/uploads
EXPOSE 8080
ENV DATA_DIR=/app/data
ENV PORT=8080
ENV TZ=Asia/Jakarta
CMD ["./app"]
