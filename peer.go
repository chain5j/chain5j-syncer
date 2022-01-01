// Package syncer
//
// @author: xwc1125
package syncer

import (
	"errors"
	"github.com/chain5j/chain5j-pkg/types"
	"github.com/chain5j/chain5j-protocol/models"
	"github.com/chain5j/chain5j-protocol/protocol"
	"sync"
)

var (
	errClosed            = errors.New("peer set is closed")
	errAlreadyRegistered = errors.New("peer is already registered")
	errNotRegistered     = errors.New("peer is not registered")
)

// 每一个节点都给启动一个节点，让其来进行同步等处理
type peer struct {
	models.P2PID
	p2p protocol.P2PService

	head        types.Hash
	blockHeight uint64
	mu          sync.RWMutex

	quitCh chan struct{}
}

func newPeer(p2p protocol.P2PService, id models.P2PID) *peer {
	return &peer{
		P2PID:       id,
		p2p:         p2p,
		blockHeight: 0,
		quitCh:      make(chan struct{}),
	}
}

func (p *peer) close() {
	close(p.quitCh)
}

func (p *peer) SetHead(hash types.Hash, height uint64) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.blockHeight > height {
		return
	}

	copy(p.head[:], hash[:])
	p.blockHeight = height
}

func (p *peer) Head() (hash types.Hash, height uint64) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	copy(hash[:], p.head[:])
	return hash, p.blockHeight
}
