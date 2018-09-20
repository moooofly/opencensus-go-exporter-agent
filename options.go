package agent

import (
	"fmt"
	"log"
	"os"
	"time"
)

const DefaultConfigPath = "/etc/podinfo/labels"

const DefaultUnixSocketEndpoint = "/var/run/hunter-agent.sock"

const (
	DefaultTCPPort uint16 = 12345
	DefaultTCPHost string = "0.0.0.0"
)

var DefaultTCPEndpoint = fmt.Sprintf("%s:%d", DefaultTCPHost, DefaultTCPPort)

// options are the options to be used when initializing the Hunter agent exporter.
type options struct {
	// Hunter agent listening address
	addrs  map[string]string
	logger *log.Logger

	// OnError is the hook to be called when there is
	// an error occurred.
	// If no custom hook is set, errors are logged.
	// Optional.
	onError func(err error)

	// bundleDelayThreshold determines the max amount of time
	// the exporter can wait before uploading data to the backend.
	// Optional.
	bundleDelayThreshold time.Duration
	// bundleCountThreshold determines how many items
	// can be buffered before batch uploading them to the backend.
	// Optional.
	bundleCountThreshold int
}

var defaultExporterOptions = options{
	addrs: map[string]string{
		"tcp": DefaultTCPEndpoint,
		//"unix": DefaultUnixSocketEndpoint,
	},
	logger:               log.New(os.Stderr, "[hunter-agent-exporter] ", log.LstdFlags),
	onError:              nil,
	bundleDelayThreshold: 2 * time.Second,
	bundleCountThreshold: 300,
}

// ExporterOption sets options such as addrs, logger, etc.
type ExporterOption func(*options)

// Logger sets the logger used to report errors.
func Addrs(addrs map[string]string) ExporterOption {
	return func(o *options) {
		for k, v := range addrs {
			o.addrs[k] = v
		}
	}
}

// Logger sets the logger used to report errors.
func Logger(logger *log.Logger) ExporterOption {
	return func(o *options) {
		o.logger = logger
	}
}

// ErrFun sets custom error function.
func ErrFun(errFun func(err error)) ExporterOption {
	return func(o *options) {
		o.onError = errFun
	}
}

// DelayThreshold sets the max amount of time the exporter can wait before uploading data to the backend
func DelayThreshold(t time.Duration) ExporterOption {
	return func(o *options) {
		o.bundleDelayThreshold = t
	}
}

// CountThreshold sets how many items can be buffered before batch uploading them to the backend
func CountThreshold(cnt int) ExporterOption {
	return func(o *options) {
		o.bundleCountThreshold = cnt
	}
}
