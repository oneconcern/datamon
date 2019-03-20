package csi

import (
	"context"
	"net"
	"net/url"
	"os"
	"sync"

	"go.uber.org/zap"

	"google.golang.org/grpc"

	"github.com/container-storage-interface/spec/lib/go/csi/v0"
)

// This is based on the server written in gcp-filestore-csi-driver

type NonBlockingGRPCServer interface {
	Start(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer, l *zap.Logger)
	Wait()
	Stop()
	ForceStop()
}

func NewNonBlockingGRPCServer() NonBlockingGRPCServer {
	return &nonBlockingGRPCServer{}
}

type nonBlockingGRPCServer struct {
	wg     sync.WaitGroup
	server *grpc.Server
}

func (s *nonBlockingGRPCServer) Start(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer, logger *zap.Logger) {

	s.wg.Add(1)

	go s.serve(endpoint, ids, cs, ns, logger)

	return
}

func (s *nonBlockingGRPCServer) Wait() {
	s.wg.Wait()
}

func (s *nonBlockingGRPCServer) Stop() {
	s.server.GracefulStop()
}

func (s *nonBlockingGRPCServer) ForceStop() {
	s.server.Stop()
}

func (s *nonBlockingGRPCServer) serve(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer, logger *zap.Logger) {
	u, err := url.Parse(endpoint)
	if err != nil {
		logger.Fatal(err.Error())
	}

	var addr string
	if u.Scheme == "unix" {
		addr = u.Path
		if err := os.Remove(addr); err != nil && !os.IsNotExist(err) {
			logger.Fatal("failed to remove", zap.String("addr", addr), zap.Error(err))
		}
	} else if u.Scheme == "tcp" {
		addr = u.Host
	} else {
		logger.Fatal("endpoint scheme not supported", zap.String("scheme", u.Scheme))
	}

	listener, err := net.Listen(u.Scheme, addr)
	if err != nil {
		logger.Fatal("failed to listen", zap.Error(err))
	}
	logger.Info("started listening", zap.String("scheme", u.Scheme), zap.String("addr", addr))

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(getInterceptor(logger)),
	}
	server := grpc.NewServer(opts...)
	s.server = server

	if ids != nil {
		csi.RegisterIdentityServer(server, ids)
	}
	if cs != nil {
		csi.RegisterControllerServer(server, cs)
	}
	if ns != nil {
		csi.RegisterNodeServer(server, ns)
	}
	logger.Info("Listening for connections", zap.Any("addr", listener))

	err = server.Serve(listener)
	if err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}

func getInterceptor(l *zap.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		l.Debug("grpc call", zap.String("method", info.FullMethod), zap.Any("req", req))
		resp, err := handler(ctx, req)
		if err != nil {
			l.Error("grpc error", zap.Error(err))
		} else {
			l.Debug("grpc response", zap.Any("resp", resp))
		}
		return resp, err
	}
}
