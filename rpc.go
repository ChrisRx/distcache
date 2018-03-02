package distcache

import (
	"context"
	"fmt"
	"net"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	pb "github.com/ChrisRx/distcache/distcachepb"
)

type RPCServer struct {
	l     net.Listener
	grpc  *grpc.Server
	cache *cache
	peers *peers
}

func NewRPCServer(addr string, opts ...grpc.ServerOption) (*RPCServer, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	r := &RPCServer{
		l:     l,
		grpc:  grpc.NewServer(opts...),
		cache: newCache(),
		peers: newPeers(l.Addr().String()),
	}
	pb.RegisterCacheServer(r.grpc, r)
	return r, nil
}

func (r *RPCServer) Addr() net.Addr { return r.l.Addr() }
func (r *RPCServer) Peers() *peers  { return r.peers }
func (r *RPCServer) Start()         { r.grpc.Serve(r.l) }
func (r *RPCServer) Stop()          { r.grpc.Stop() }

// Trxn determines the locality of the cache transaction by performing a
// consistent hash over the requested key. If the key is not on the local
// instance a grpc call to the remote server is made.
func (r *RPCServer) Trxn(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	addr := r.peers.Get(req.Key)
	localAddr := r.l.Addr().String()
	if addr != localAddr {
		conn, err := grpc.Dial(addr, grpc.WithInsecure())
		if err != nil {
			return nil, err
		}
		client := pb.NewCacheClient(conn)
		return client.Trxn(ctx, req)
	}
	return r.trxn(ctx, req)
}

// trxn performs a cache transaction on the local server cache only.
func (r *RPCServer) trxn(ctx context.Context, req *pb.Request) (*pb.Response, error) {
	resp := &pb.Response{
		Status: pb.Status_Ok,
		Server: r.l.Addr().String(),
		Key:    req.Key,
	}
	switch req.Type {
	case pb.TrxnType_GET:
		val, ok := r.cache.Get(req.Key)
		if !ok {
			resp.Status = pb.Status_Err
			resp.Value = fmt.Sprintf("cannot find key: %#v", req.Key)
			return resp, nil
		}
		resp.Value = val
		return resp, nil
	case pb.TrxnType_SET:
		r.cache.Set(req.Key, req.Value)
		return resp, nil
	case pb.TrxnType_DELETE:
		r.cache.Delete(req.Key)
		return resp, nil
	}
	return nil, errors.Errorf("invalid TrxnType: %#v", req.Type)
}
