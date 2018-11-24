package sync

import "sync"

type NodeList struct {
	addrs   []string
	addrsMu sync.RWMutex
}

func (l *NodeList) SetLatestNodeAddrs(addrs []string) {
	if len(addrs) <= 0 {
		return
	}
	l.addrsMu.Lock()
	l.addrs = addrs
	l.addrsMu.Unlock()
}
func (l *NodeList) NodeAddrs() []string {
	l.addrsMu.RLock()
	defer l.addrsMu.RUnlock()
	return l.addrs
}
