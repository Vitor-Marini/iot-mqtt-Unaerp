# ---------- build ----------
FROM golang:1.21-alpine AS build

WORKDIR /src

# Baixa as dependências numa camada própria: só invalida quando go.mod/go.sum
# mudam, não a cada edição de código.
COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# Binário estático, sem informação de debug, para caber na imagem final.
RUN CGO_ENABLED=0 GOOS=linux go build \
        -ldflags="-s -w" \
        -o /out/subscriber ./cmd/subscriber

# ---------- runtime ----------
FROM alpine:3.20

# Certificados raiz: necessários caso o broker passe a usar TLS (porta 8883).
RUN apk add --no-cache ca-certificates \
    && adduser -D -u 10001 subscriber

COPY --from=build /out/subscriber /usr/local/bin/subscriber

USER subscriber
EXPOSE 2112

ENTRYPOINT ["/usr/local/bin/subscriber"]
