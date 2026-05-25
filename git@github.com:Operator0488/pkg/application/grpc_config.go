package application

import (
	"strings"
	"time"
)

const (
	// private server
	envGrpcPrivateServerPort              = "grpc.server.private.port"
	envGrpcPrivateServerHost              = "grpc.server.private.host"
	envGrpcPrivateServerMaxRecvMsgSize    = "grpc.server.private.max_recv_msg_size"
	envGrpcPrivateServerMaxSendMsgSize    = "grpc.server.private.max_send_msg_size"
	envGrpcPrivateServerConnectionTimeout = "grpc.server.private.connection_timeout"
	envGrpcPrivateServerKeepAliveTime     = "grpc.server.private.keepalive_time"
	envGrpcPrivateServerKeepAliveTimeout  = "grpc.server.private.keepalive_timeout"

	// public server
	envGrpcPublicServerPort              = "grpc.server.public.port"
	envGrpcPublicServerHost              = "grpc.server.public.host"
	envGrpcPublicServerMaxRecvMsgSize    = "grpc.server.public.max_recv_msg_size"
	envGrpcPublicServerMaxSendMsgSize    = "grpc.server.public.max_send_msg_size"
	envGrpcPublicServerConnectionTimeout = "grpc.server.public.connection_timeout"
	envGrpcPublicServerKeepAliveTime     = "grpc.server.public.keepalive_time"
	envGrpcPublicServerKeepAliveTimeout  = "grpc.server.public.keepalive_timeout"

	// gRPC Client
	grpcClientPrefix    = "grpc.client."
	cfgAddress          = ".address"
	cfgTimeout          = ".timeout"
	cfgMaxRecvMsgSize   = ".max_recv_msg_size"
	cfgMaxSendMsgSize   = ".max_send_msg_size"
	cfgKeepAliveTime    = ".keepalive_time"
	cfgKeepAliveTimeout = ".keepalive_timeout"

	// defaults
	defaultGrpcPrivatePort       = "50051"
	defaultGrpcPublicPort        = "50052"
	defaultGrpcMaxRecvMsgSize    = 4 * 1024 * 1024 // 4MB
	defaultGrpcMaxSendMsgSize    = 4 * 1024 * 1024 // 4MB
	defaultGrpcConnectionTimeout = 120 * time.Second
	defaultGrpcKeepAliveTime     = 30 * time.Second
	defaultGrpcKeepAliveTimeout  = 10 * time.Second
	defaultGrpcClientTimeout     = 30 * time.Second
)

type grpcServerConfig struct {
	Host              string
	Port              string
	MaxRecvMsgSize    int
	MaxSendMsgSize    int
	ConnectionTimeout time.Duration
	KeepAliveTime     time.Duration
	KeepAliveTimeout  time.Duration
}

func (g *grpcServerConfig) GetAddr() string {
	if g.Host == "" {
		return ":" + g.Port
	}
	return strings.Join([]string{g.Host, g.Port}, ":")
}

func (a *appConfig) GetGrpcPrivateServerConfig() grpcServerConfig {
	return grpcServerConfig{
		Host:              getStringOrDefault(a.GetString(envGrpcPrivateServerHost), ""),
		Port:              getStringOrDefault(a.GetString(envGrpcPrivateServerPort), defaultGrpcPrivatePort),
		MaxRecvMsgSize:    getIntOrDefault(a.GetInt(envGrpcPrivateServerMaxRecvMsgSize), defaultGrpcMaxRecvMsgSize),
		MaxSendMsgSize:    getIntOrDefault(a.GetInt(envGrpcPrivateServerMaxSendMsgSize), defaultGrpcMaxSendMsgSize),
		ConnectionTimeout: getDurationOrDefault(a.GetDuration(envGrpcPrivateServerConnectionTimeout), defaultGrpcConnectionTimeout),
		KeepAliveTime:     getDurationOrDefault(a.GetDuration(envGrpcPrivateServerKeepAliveTime), defaultGrpcKeepAliveTime),
		KeepAliveTimeout:  getDurationOrDefault(a.GetDuration(envGrpcPrivateServerKeepAliveTimeout), defaultGrpcKeepAliveTimeout),
	}
}

func (a *appConfig) GetGrpcPublicServerConfig() grpcServerConfig {
	return grpcServerConfig{
		Host:              getStringOrDefault(a.GetString(envGrpcPublicServerHost), ""),
		Port:              getStringOrDefault(a.GetString(envGrpcPublicServerPort), defaultGrpcPublicPort),
		MaxRecvMsgSize:    getIntOrDefault(a.GetInt(envGrpcPublicServerMaxRecvMsgSize), defaultGrpcMaxRecvMsgSize),
		MaxSendMsgSize:    getIntOrDefault(a.GetInt(envGrpcPublicServerMaxSendMsgSize), defaultGrpcMaxSendMsgSize),
		ConnectionTimeout: getDurationOrDefault(a.GetDuration(envGrpcPublicServerConnectionTimeout), defaultGrpcConnectionTimeout),
		KeepAliveTime:     getDurationOrDefault(a.GetDuration(envGrpcPublicServerKeepAliveTime), defaultGrpcKeepAliveTime),
		KeepAliveTimeout:  getDurationOrDefault(a.GetDuration(envGrpcPublicServerKeepAliveTimeout), defaultGrpcKeepAliveTimeout),
	}
}

type grpcClientConfig struct {
	Address          string
	Timeout          time.Duration
	MaxRecvMsgSize   int
	MaxSendMsgSize   int
	KeepAliveTime    time.Duration
	KeepAliveTimeout time.Duration
}

func (a *appConfig) GetGrpcClientConfig(serviceName string) grpcClientConfig {
	base := serviceName
	if !strings.HasPrefix(base, grpcClientPrefix) {
		base = grpcClientPrefix + base
	}

	return grpcClientConfig{
		Address:          a.GetString(base + cfgAddress),
		Timeout:          getDurationOrDefault(a.GetDuration(base+cfgTimeout), defaultGrpcClientTimeout),
		MaxRecvMsgSize:   getIntOrDefault(a.GetInt(base+cfgMaxRecvMsgSize), defaultGrpcMaxRecvMsgSize),
		MaxSendMsgSize:   getIntOrDefault(a.GetInt(base+cfgMaxSendMsgSize), defaultGrpcMaxSendMsgSize),
		KeepAliveTime:    getDurationOrDefault(a.GetDuration(base+cfgKeepAliveTime), defaultGrpcKeepAliveTime),
		KeepAliveTimeout: getDurationOrDefault(a.GetDuration(base+cfgKeepAliveTimeout), defaultGrpcKeepAliveTimeout),
	}
}

func getIntOrDefault(value int, def int) int {
	if value == 0 {
		return def
	}
	return value
}

func getDurationOrDefault(value time.Duration, def time.Duration) time.Duration {
	if value == 0 {
		return def
	}
	return value
}
