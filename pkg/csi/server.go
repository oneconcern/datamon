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
// This is essentially a wrapper around the grpc server and setups up the right profile for receiving grpc requests.

type NonBlockingGRPCServer interface {
	Start(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer)
	Wait()
}

func NewNonBlockingGRPCServer(logger *zap.Logger) NonBlockingGRPCServer {
	return &nonBlockingGRPCServer{
		logger: logger,
	}
}

type nonBlockingGRPCServer struct {
	wg     sync.WaitGroup
	server *grpc.Server
	logger *zap.Logger
}

func (s *nonBlockingGRPCServer) Start(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer) {
	s.wg.Add(1)
	go s.serve(endpoint, ids, cs, ns)
}

func (s *nonBlockingGRPCServer) Wait() {
	s.wg.Wait()
}

func (s *nonBlockingGRPCServer) serve(endpoint string, ids csi.IdentityServer, cs csi.ControllerServer, ns csi.NodeServer) {
	u, err := url.Parse(endpoint)
	if err != nil {
		s.logger.Fatal(err.Error())
		return
	}

	var addr string
	switch u.Scheme {
	case "unix":
		addr = u.Path
		if err = os.Remove(addr); err != nil && !os.IsNotExist(err) {
			s.logger.Fatal("failed to remove", zap.String("addr", addr), zap.Error(err))
		}
	case "tcp":
		addr = u.Host
	default:
		s.logger.Fatal("endpoint scheme not supported", zap.String("scheme", u.Scheme))
	}

	listener, err := net.Listen(u.Scheme, addr)
	if err != nil {
		s.logger.Fatal("failed to listen", zap.Error(err))
	}
	s.logger.Info("started listening", zap.String("scheme", u.Scheme), zap.String("addr", addr))

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(getInterceptor(s.logger)),
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
	s.logger.Info("Listening for connections", zap.Any("addr", listener))

	err = server.Serve(listener)
	if err != nil {
		s.logger.Fatal("Failed to start server", zap.Error(err))
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
