// Package syncer
//
// @author: xwc1125
package syncer

import (
	"github.com/chain5j/chain5j-protocol/models"
	"sync"
)

type peerSet struct {
	peers  map[models.P2PID]*peer
	lock   sync.RWMutex
	closed bool
}

func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[models.P2PID]*peer),
	}
}

func (ps *peerSet) Register(p *peer, goFunc func()) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if ps.closed {
		return errClosed
	}
	if _, ok := ps.peers[p.P2PID]; ok {
		return errAlreadyRegistered
	}
	ps.peers[p.P2PID] = p
	// go p.broadcast()
	if goFunc != nil {
		go goFunc()
	}

	return nil
}

func (ps *peerSet) Deregister(id models.P2PID) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	p, ok := ps.peers[id]
	if !ok {
		return errNotRegistered
	}
	delete(ps.peers, id)
	p.close()

	return nil
}

func (ps *peerSet) Peer(id models.P2PID) *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	return ps.peers[id]
}

func (ps *peerSet) IsExist(id models.P2PID) bool {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	p := ps.peers[id]
	if p != nil {
		return true
	}
	return false
}

func (ps *peerSet) Close() {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	for _, p := range ps.peers {
		p.close()
	}
	ps.closed = true
}

// 强制执行同步时，选择对方高度最高的进行同步
func (ps *peerSet) BestPeer() *peer {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	var (
		bestPeer   *peer
		bestHeight uint64
	)
	for _, p := range ps.peers {
		if _, height := p.Head(); bestPeer == nil || height > bestHeight {
			bestPeer, bestHeight = p, height
		}
	}
	return bestPeer
}
