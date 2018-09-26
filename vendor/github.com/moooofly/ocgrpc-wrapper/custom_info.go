package wrapper

import "go.opencensus.io/trace"

// Ref: https://github.com/yancl/hunter-spec/blob/master/spec/trace.md

type CustomInfo struct {
	ServiceName string // cluster name of k8s pod from label
	HostName    string // hostname of pod or instance

	// TODO: not need at present
	//RemoteAddr  string
}

type AdditionalInfo struct {
	UID int64
}

type ServerCustomInfo struct {
	CustomInfo

	Kind string // what kind of service type the server belongs to (http/grpc/job)
}

type ClientCustomInfo struct {
	CustomInfo

	RemoteKind string // what kind of service type the client is calling on (http/grpc/mysql/redis)
	Sampler    trace.Sampler
}

func NewServerCustomInfo(svcName, hostname string) *ServerCustomInfo {
	return &ServerCustomInfo{
		CustomInfo{
			svcName,
			hostname,
		},
		"grpc",
	}
}
func NewClientCustomInfo(svcName, hostname string, sp trace.Sampler) *ClientCustomInfo {
	return &ClientCustomInfo{
		CustomInfo{
			svcName,
			hostname,
		},
		"grpc",
		sp,
	}
}
