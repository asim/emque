package selector

import (
	"errors"
	"hash/crc32"
	"sync"
)

// All is a Selector that returns all servers
type All struct {
	sync.RWMutex
	servers []string
}

// Shard is a Selector that shards to a single server
type Shard struct {
	sync.RWMutex
	servers []string
}

func (sa *All) Get(topic string) ([]string, error) {
	sa.RLock()
	if len(sa.servers) == 0 {
		sa.RUnlock()
		return nil, errors.New("no servers")
	}
	servers := sa.servers
	sa.RUnlock()
	return servers, nil
}

func (sa *All) Set(servers ...string) error {
	sa.Lock()
	sa.servers = servers
	sa.Unlock()
	return nil
}

func (ss *Shard) Get(topic string) ([]string, error) {
	ss.RLock()
	length := len(ss.servers)
	if length == 0 {
		ss.RUnlock()
		return nil, errors.New("no servers")
	}
	if length == 1 {
		servers := ss.servers
		ss.RUnlock()
		return servers, nil
	}
	cs := crc32.ChecksumIEEE([]byte(topic))
	server := ss.servers[cs%uint32(length)]
	ss.RUnlock()
	return []string{server}, nil
}

func (ss *Shard) Set(servers ...string) error {
	ss.Lock()
	ss.servers = servers
	ss.Unlock()
	return nil
}
