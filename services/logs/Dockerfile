FROM golang:1.27.1-alpine3.24 AS build

WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /logs ./services/logs/cmd/server

FROM alpine:3.24

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /logs /app/logs
COPY services/logs/config/local.yaml /app/config/local.yaml

USER 10001:10001
ENTRYPOINT ["/app/logs"]
CMD ["-config", "/app/config/local.yaml"]
