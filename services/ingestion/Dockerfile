# Build context must be the repository root (the service imports contracts).
FROM golang:1.27 AS build
WORKDIR /src
COPY contracts/ ./contracts/
COPY services/ingestion/ ./services/ingestion/
WORKDIR /src/services/ingestion
ENV GOWORK=off
RUN CGO_ENABLED=0 go build -mod=readonly -trimpath -o /out/ingestion ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/ingestion /ingestion
ENTRYPOINT ["/ingestion"]
CMD ["-config", "/etc/logflux/config.yaml"]
