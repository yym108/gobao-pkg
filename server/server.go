// Package server 提供统一的服务启动模板，
// 同时运行 gRPC（业务）+ HTTP（healthz、metrics）并支持优雅关停。
package server

import (
	"context"
	"errors"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// Options 描述服务的启动配置
type Options struct {
	HTTPAddr string             // HTTP 监听地址，如 ":8080"，"127.0.0.1:0" 表示自动分配
	GRPCAddr string             // gRPC 监听地址
	Register func(*grpc.Server) // 业务 gRPC 服务注册回调，可为 nil
}

// Server 同时持有 gRPC 和 HTTP 两个服务实例
type Server struct {
	name   string
	opts   Options
	grpc   *grpc.Server
	http   *http.Server
	httpLn net.Listener
	grpcLn net.Listener
}

// New 构造 Server，此时不监听端口，调用 Run 后才启动。
func New(name string, opts Options) *Server {
	return &Server{name: name, opts: opts}
}

// HTTPListenAddr 返回 HTTP 实际监听地址（端口为 0 时用于获取自动分配的端口）。
// Run 尚未执行时返回空字符串。
func (s *Server) HTTPListenAddr() string {
	if s.httpLn == nil {
		return ""
	}
	return s.httpLn.Addr().String()
}

// GRPCListenAddr 返回 gRPC 实际监听地址。
// Run 尚未执行时返回空字符串。
func (s *Server) GRPCListenAddr() string {
	if s.grpcLn == nil {
		return ""
	}
	return s.grpcLn.Addr().String()
}

// Run 启动 gRPC + HTTP 并阻塞，直到 ctx 取消（优雅关停）或某个服务出错。
func (s *Server) Run(ctx context.Context) error {
	var err error
	s.grpcLn, err = net.Listen("tcp", s.opts.GRPCAddr)
	if err != nil {
		return err
	}
	s.httpLn, err = net.Listen("tcp", s.opts.HTTPAddr)
	if err != nil {
		return err
	}

	s.grpc = grpc.NewServer()
	healthpb.RegisterHealthServer(s.grpc, health.NewServer())
	if s.opts.Register != nil {
		s.opts.Register(s.grpc)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/metrics", promhttp.Handler())
	s.http = &http.Server{Handler: mux}

	errCh := make(chan error, 2)
	go func() { errCh <- s.grpc.Serve(s.grpcLn) }()
	go func() { errCh <- s.http.Serve(s.httpLn) }()

	select {
	case <-ctx.Done():
		s.grpc.GracefulStop()
		_ = s.http.Shutdown(context.Background())
		return nil
	case e := <-errCh:
		s.grpc.GracefulStop()
		_ = s.http.Shutdown(context.Background())
		if errors.Is(e, http.ErrServerClosed) || e == grpc.ErrServerStopped {
			return nil
		}
		return e
	}
}
