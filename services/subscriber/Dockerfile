# ---------- build ----------
FROM golang:1.26-alpine AS build

WORKDIR /src

# Dependencias numa camada propria: so invalida quando go.mod/go.sum mudam,
# nao a cada edicao de codigo.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Pacote unico na raiz do modulo (layout plano), binario estatico sem simbolos.
RUN CGO_ENABLED=0 GOOS=linux go build \
        -ldflags="-s -w" \
        -o /out/subscriber .

# ---------- runtime ----------
FROM alpine:3.20

# Certificados raiz: necessarios se o InfluxDB estiver atras de HTTPS.
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 subscriber

COPY --from=build /out/subscriber /usr/local/bin/subscriber

USER subscriber

# Sem EXPOSE: este servico nao serve HTTP. Ele escreve no InfluxDB.
ENTRYPOINT ["/usr/local/bin/subscriber"]
