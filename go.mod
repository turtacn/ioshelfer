module github.com/turtacn/ioshelfer

go 1.20

require (
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.22.0
	go.uber.org/zap v1.27.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.62.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/cobra v1.9.1 // indirect
	github.com/spf13/viper v1.20.1 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
)

replace (
	golang.org/x/sys => github.com/golang/sys v0.30.0
	google.golang.org/protobuf => github.com/protocolbuffers/protobuf-go v1.36.5
)
