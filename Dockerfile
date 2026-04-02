FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.version=${VERSION:-dev}" \
    -o /bin/server \
    ./cmd/server

FROM gcr.io/distroless/static-debian12 AS runtime

COPY --from=builder /bin/server /server

EXPOSE 8080

ENTRYPOINT ["/server", "serve"]
