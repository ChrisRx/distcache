package distcache

import "github.com/golang/groupcache/consistenthash"

type peers struct {
	ring  *consistenthash.Map
	peers []string
}

func newPeers(addr string) *peers {
	p := &peers{}
	p.Set(addr)
	return p
}

func (p *peers) Get(key string) string {
	return p.ring.Get(key)
}

func (p *peers) Set(peers ...string) {
	p.peers = peers
	p.ring = consistenthash.New(1, nil)
	p.ring.Add(p.peers...)
}
