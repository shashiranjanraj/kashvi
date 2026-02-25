// Package grpc provides a production-ready gRPC server for Kashvi.
//
// Features:
//   - Panic-recovery interceptor (returns INTERNAL status instead of killing goroutine)
//   - Request logging interceptor (method, duration, status code)
//   - Prometheus metrics interceptor (grpc_server_handled_total, grpc_server_handling_seconds)
//   - Standard gRPC health-check service (grpc.health.v1.Health)
//   - Graceful shutdown via Stop()
//
// Usage in server bootstrap:
//
//	grpcSrv, lis, err := grpc.Start(config.GRPCPort())
//	// ...run until signal...
//	grpc.Stop(grpcSrv)
package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ─── Prometheus metrics ───────────────────────────────────────────────────────

var (
	grpcRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "grpc_server_handled_total",
		Help: "Total number of gRPC calls completed by method and code.",
	}, []string{"grpc_method", "grpc_code"})

	grpcRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "grpc_server_handling_seconds",
		Help:    "Histogram of gRPC response latency in seconds.",
		Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
	}, []string{"grpc_method"})
)

// ─── Interceptors ─────────────────────────────────────────────────────────────

// recoveryInterceptor catches panics in gRPC handlers and returns a gRPC
// INTERNAL error instead of crashing the process.
func recoveryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("grpc: panic recovered",
				"method", info.FullMethod,
				"panic", r,
				"stack", string(debug.Stack()),
			)
			err = status.Errorf(codes.Internal, "internal server error")
		}
	}()
	return handler(ctx, req)
}

// loggingInterceptor logs each unary RPC call with its duration and result.
func loggingInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	dur := time.Since(start)

	code := codes.OK
	if err != nil {
		code = status.Code(err)
	}

	slog.Info("grpc: request",
		"method", info.FullMethod,
		"duration_ms", dur.Milliseconds(),
		"code", code.String(),
	)
	return resp, err
}

// metricsInterceptor records Prometheus counters and histograms per RPC.
func metricsInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()
	resp, err := handler(ctx, req)
	dur := time.Since(start)

	code := codes.OK
	if err != nil {
		code = status.Code(err)
	}

	grpcRequestsTotal.WithLabelValues(info.FullMethod, code.String()).Inc()
	grpcRequestDuration.WithLabelValues(info.FullMethod).Observe(dur.Seconds())
	return resp, err
}

// chainUnary chains multiple UnaryServerInterceptors into one.
// They execute in order: interceptors[0] wraps interceptors[1] wraps … handler.
func chainUnary(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			i := i
			next := chain
			chain = func(ctx context.Context, req interface{}) (interface{}, error) {
				return interceptors[i](ctx, req, info, next)
			}
		}
		return chain(ctx, req)
	}
}

// ─── Health service ───────────────────────────────────────────────────────────

// healthServer implements grpc_health_v1.HealthServer.
type healthServer struct {
	grpc_health_v1.UnimplementedHealthServer
}

func (h *healthServer) Check(
	_ context.Context,
	req *grpc_health_v1.HealthCheckRequest,
) (*grpc_health_v1.HealthCheckResponse, error) {
	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

func (h *healthServer) Watch(
	req *grpc_health_v1.HealthCheckRequest,
	stream grpc_health_v1.Health_WatchServer,
) error {
	return stream.Send(&grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	})
}

// ─── Public API ───────────────────────────────────────────────────────────────

// Start creates and starts a gRPC server on the given port.
// Returns the server and the net.Listener so callers can gracefully stop it.
func Start(port string) (*grpc.Server, net.Listener, error) {
	addr := ":" + port

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("grpc: listen on %s: %w", addr, err)
	}

	srv := grpc.NewServer(
		grpc.UnaryInterceptor(
			chainUnary(
				recoveryInterceptor,
				loggingInterceptor,
				metricsInterceptor,
			),
		),
		// Connection settings for high throughput.
		grpc.MaxRecvMsgSize(4*1024*1024), // 4 MB
		grpc.MaxSendMsgSize(4*1024*1024), // 4 MB
	)

	// Register standard health service.
	grpc_health_v1.RegisterHealthServer(srv, &healthServer{})

	// Enable server reflection so tools like grpcurl work without proto files.
	reflection.Register(srv)

	slog.Info("gRPC server starting", "addr", addr)

	go func() {
		if err := srv.Serve(lis); err != nil {
			slog.Error("grpc: serve error", "error", err)
		}
	}()

	return srv, lis, nil
}

// Stop gracefully shuts down the gRPC server, waiting for in-flight RPCs to
// complete.
func Stop(srv *grpc.Server) {
	if srv == nil {
		return
	}
	slog.Info("gRPC server shutting down")
	srv.GracefulStop()
}
