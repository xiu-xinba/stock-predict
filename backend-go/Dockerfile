FROM golang:1.26.4-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM alpine:3.22

RUN apk add --no-cache ca-certificates postgresql-client \
    && addgroup -S app \
    && adduser -S -G app app
WORKDIR /app
COPY --from=build /out/api /usr/local/bin/stock-api
COPY --from=build /out/migrate /usr/local/bin/stock-migrate
COPY docker/postgres/init/01-runtime-role.sh /usr/local/bin/stock-grant-runtime
RUN chmod 0755 /usr/local/bin/stock-grant-runtime
USER app
EXPOSE 5070
ENTRYPOINT ["stock-api"]
