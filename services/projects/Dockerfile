FROM golang:1.27.1-alpine3.24 AS build

WORKDIR /src
COPY contracts/ ./contracts/
COPY services/projects/ ./services/projects/
WORKDIR /src/services/projects
ENV GOWORK=off
RUN CGO_ENABLED=0 go build -mod=readonly -trimpath -o /out/projects ./cmd/server

FROM alpine:3.24

RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /out/projects /app/projects
COPY services/projects/config/config.yaml /app/config/config.yaml

USER 10001:10001
ENTRYPOINT ["/app/projects"]
CMD ["-config", "/app/config/config.yaml"]
