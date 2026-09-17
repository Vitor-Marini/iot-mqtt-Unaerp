# ---------- build ----------
FROM golang:1.24-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
        -ldflags="-s -w" \
        -o /out/ota-server .

# ---------- runtime ----------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata \
    && adduser -D -u 10001 otasrv

WORKDIR /app

COPY --from=build /out/ota-server /usr/local/bin/ota-server

RUN mkdir -p /app/storage && chown -R otasrv:otasrv /app

USER otasrv

EXPOSE 8080

VOLUME ["/app/storage"]

ENTRYPOINT ["/usr/local/bin/ota-server"]
