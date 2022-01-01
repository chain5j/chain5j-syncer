// Package syncer
//
// @author: xwc1125
package syncer

import (
	"context"
	"github.com/chain5j/chain5j-protocol/models"
	"github.com/chain5j/chain5j-protocol/protocol"
	"github.com/chain5j/logger"
	"time"
)

var (
	_ protocol.Syncer = new(syncer)
)

const (
	forceSyncCycle       = 10 * time.Second // 强制同步的间隔时间
	forceSyncHeaderCycle = 3 * time.Minute  // 强制同步Header的间隔时间
)

type syncer struct {
	log    logger.Logger
	ctx    context.Context
	cancel context.CancelFunc

	p2p       protocol.P2PService
	apps      protocol.Apps
	blockRW   protocol.BlockReadWriter
	handshake protocol.Handshake

	handshakePeerCh  chan *models.HandshakeMsg
	blockCompletedCh chan struct{}
	peers            *peerSet

	queues map[uint64]*peerBlock // height==>Block
	// knownHashes map[string]uint64     // hash==>height

	syncDrop *time.Timer // Timed connection dropper if sync progress isn't validated in time

	// checkpointNumber uint64     // Block number for the sync progress validator to cross reference
	// checkpointHash   types.Hash // Block hash for the sync progress validator to cross reference

	quitCh chan struct{}
}

func NewSyncer(rootCtx context.Context, opts ...option) (protocol.Syncer, error) {
	ctx, cancel := context.WithCancel(rootCtx)
	s := &syncer{
		log:    logger.New("syncer"),
		ctx:    ctx,
		cancel: cancel,

		handshakePeerCh:  make(chan *models.HandshakeMsg),
		blockCompletedCh: make(chan struct{}),

		queues: make(map[uint64]*peerBlock),
		// knownHashes: make(map[string]uint64),

		peers:  newPeerSet(),
		quitCh: make(chan struct{}),
	}
	if err := apply(s, opts...); err != nil {
		s.log.Error("apply is error", "err", err)
		return nil, err
	}
	return s, nil
}

func (s *syncer) Start() error {
	go s.syncBlocks()
	go s.listen()
	return nil
}

func (s *syncer) Stop() error {
	close(s.quitCh)
	// close(s.handshakePeerCh)
	return nil
}

func (s *syncer) syncBlocks() {

	// 远程的节点断开
	dropPeerCh := make(chan models.P2PID)
	dropPeerSub := s.p2p.SubscribeDropPeer(dropPeerCh)

	// 协议订阅
	handshakePeerSub := s.handshake.SubscribeHandshake(s.handshakePeerCh)

	forceSync := time.NewTicker(forceSyncCycle)
	defer forceSync.Stop()
	forceSyncHeader := time.NewTicker(forceSyncHeaderCycle)
	defer forceSyncHeader.Stop()

	for {
		select {
		case ch := <-dropPeerCh:
			s.peers.Deregister(ch)
		case err := <-dropPeerSub.Err():
			s.log.Error("dropPeerSub", "err", err)
		case ch := <-s.handshakePeerCh:
			// TODO 有新的节点进来，就会直接调用同步接口
			peer := s.peers.Peer(ch.Peer)
			if peer != nil {
				peer.SetHead(ch.CurrentBlockHash, ch.CurrentBlockHeight)
				break
			}
			peer = newPeer(s.p2p, ch.Peer)
			s.peers.Register(peer, nil)

			peer.SetHead(ch.CurrentBlockHash, ch.CurrentBlockHeight)

			go s.syncBlocksLoop(peer)
		case err := <-handshakePeerSub.Err():
			s.log.Error("handshakePeerSub", "err", err)
		case <-s.blockCompletedCh:
			s.blockCompleted()
		case <-forceSync.C:
			// 强制执行同步时，选择对方高度最高的进行同步
			go s.syncBlocksLoop(s.peers.BestPeer())
		case <-forceSyncHeader.C:
			go s.syncBlocksHeaderLoop(s.peers)
		case <-s.quitCh:
			return
		}
	}
}

func (s *syncer) syncBlocksLoop(peer *peer) {
	if peer == nil {
		return
	}

	currentBlock := s.blockRW.CurrentBlock()
	_, pHeight := peer.Head()
	if pHeight <= currentBlock.Height() {
		return
	}
	// 开始进行同步下载
	// TODO 计算local height和远程的相隔，然后获取其间隔，如果相隔较近
	s.findAncestor(peer, pHeight)

}

// 每10分钟进行一次全局的peer的handshake
func (s *syncer) syncBlocksHeaderLoop(peers *peerSet) {
	if peers == nil || len(peers.peers) == 0 {
		return
	}
	for _, peer := range peers.peers {
		if peer == nil {
			return
		}
		go s.handshake.RequestHandshake(peer.P2PID)
	}
}

func (s *syncer) blockCompleted() {
	block := s.blockRW.CurrentBlock()
	next := block.Height()
	for {
		next = next + 1
		qBlock := s.queues[next]
		if qBlock == nil {
			return
		}
		if qBlock.syncing {
			return
		}
		delete(s.queues, next)
		s.log.Debug("blockCompleted", "height", next)
		s.blockRW.ProcessBlock(qBlock.Block, false)
	}
}
