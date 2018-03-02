package distcache_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"google.golang.org/grpc"

	"github.com/ChrisRx/distcache"
	pb "github.com/ChrisRx/distcache/distcachepb"
)

func init() {
	http.DefaultTransport = &http.Transport{Proxy: nil}
}

func TestRPCServer(t *testing.T) {
	cases := []struct {
		Key, Value string
	}{
		{"test1", "value1"},
		{"test2", "value2"},
		{"test3", "value3"},
	}

	addrs := []string{"localhost:50501", "localhost:50502"}
	servers := make([]*distcache.RPCServer, 0)
	for _, addr := range addrs {
		s, err := distcache.NewRPCServer(addr)
		if err != nil {
			t.Fatal(err)
		}
		servers = append(servers, s)
		go s.Start()
	}
	peers := make([]string, 0)
	for _, s := range servers {
		peers = append(peers, s.Addr().String())
	}
	servers[0].Peers().Set(peers...)
	conn, err := grpc.Dial(addrs[0], grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	client := pb.NewCacheClient(conn)
	for _, tc := range cases {
		_, err := client.Trxn(context.Background(), &pb.Request{
			Type:  pb.TrxnType_SET,
			Key:   tc.Key,
			Value: tc.Value,
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	rservers := make(map[string]bool)
	for i, tc := range cases {
		resp, err := client.Trxn(context.Background(), &pb.Request{
			Type: pb.TrxnType_GET,
			Key:  tc.Key,
		})
		if err != nil {
			t.Fatal(err)
		}
		rservers[resp.Server] = true
		if diff := cmp.Diff(tc.Value, resp.Value); diff != "" {
			t.Errorf("TestCase %d: after client.Trxn: (-got +want)\n%s", i, diff)
		}
	}
	for _, p := range peers {
		if _, ok := rservers[p]; !ok {
			t.Errorf("peer did not receive at least one key/value: %#v", p)
		}
	}
}
