module LogFlux/services/ingestion

go 1.27

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/klauspost/compress v1.15.9 // indirect
	github.com/pierrec/lz4/v4 v4.1.15 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
)

require (
	LogFlux/contracts v0.0.0
	github.com/redis/go-redis/v9 v9.22.0
	github.com/segmentio/kafka-go v0.4.51
	go.yaml.in/yaml/v4 v4.0.0-rc.6
)

replace LogFlux/contracts => ../../contracts
