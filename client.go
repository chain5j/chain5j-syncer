// Package syncer
//
// @author: xwc1125
package syncer

import (
	"github.com/chain5j/chain5j-pkg/codec"
	"github.com/chain5j/chain5j-pkg/types"
	"github.com/chain5j/chain5j-protocol/models"
	"github.com/chain5j/chain5j-protocol/models/ext"
)

func (s *syncer) RequestOneHeader(peerId models.P2PID, hash types.Hash) error {
	s.log.Debug("Fetching single header", "hash", hash)
	req := &ext.GetBlockHeadersData{
		Origin: ext.HashOrNumber{
			Hash: hash,
		},
		Amount: uint64(1),
		// Skip:    uint64(0),
		Reverse: false,
	}
	bytes, err := codec.Coder().Encode(req)
	if err != nil {
		return err
	}

	return s.p2p.Send(peerId, &models.P2PMessage{
		Type: GetBlockHeadersMsg,
		Peer: "",
		Data: bytes,
	})
}

func (s *syncer) RequestHeadersByHash(peerId models.P2PID, origin types.Hash, amount int, skip int, reverse bool) error {
	s.log.Debug("Fetching batch of headers", "count", amount, "fromhash", origin, "skip", skip, "reverse", reverse)
	req := &ext.GetBlockHeadersData{
		Origin: ext.HashOrNumber{
			Hash: origin,
		},
		Amount: uint64(amount),
		// Skip:    uint64(skip),
		Reverse: reverse,
	}
	bytes, err := codec.Coder().Encode(req)
	if err != nil {
		return err
	}

	return s.p2p.Send(peerId, &models.P2PMessage{
		Type: GetBlockHeadersMsg,
		Peer: "",
		Data: bytes,
	})
}

// RequestHeadersByNumber fetches a batch of blocks' headers corresponding to the
// specified header query, based on the number of an origin block.
func (s *syncer) RequestHeadersByNumber(peerId models.P2PID, origin uint64, amount int, skip int, reverse bool) error {
	s.log.Debug("Fetching batch of headers", "count", amount, "fromnum", origin, "skip", skip, "reverse", reverse)
	req := &ext.GetBlockHeadersData{
		Origin: ext.HashOrNumber{
			Number: origin,
		},
		Amount: uint64(amount),
		// Skip:    uint64(skip),
		Reverse: reverse,
	}
	bytes, err := codec.Coder().Encode(req)
	if err != nil {
		return err
	}

	return s.p2p.Send(peerId, &models.P2PMessage{
		Type: GetBlockHeadersMsg,
		Peer: "",
		Data: bytes,
	})
}

// ====================body==============
// hashes: blockHash
func (s *syncer) RequestBlockBodies(peerId models.P2PID, hashes []types.Hash) error {
	s.log.Debug("Fetching batch of block bodies", "count", len(hashes))
	bytes, err := codec.Coder().Encode(hashes)
	if err != nil {
		return err
	}
	return s.p2p.Send(peerId, &models.P2PMessage{
		Type: GetBlockBodiesMsg,
		Peer: "",
		Data: bytes,
	})
}

// 处理区块Header
// 来自remote的header，需要进行处理
func (s *syncer) HandleBlockHeadersMsg(peerId models.P2PID, headers []*models.Header) error {
	// If no headers were received, but we're expencting a checkpoint header, consider it that
	if len(headers) == 0 && s.syncDrop != nil {
		// Stop the timer either way, decide later to drop or not
		s.syncDrop.Stop()
		s.syncDrop = nil
	}

	var hashes []types.Hash
	for _, header := range headers {
		hashes = append(hashes, header.Hash())
		s.headerHandle(peerId, header)
	}
	return s.RequestBlockBodies(peerId, hashes)
}

// 处理Bodies
func (s *syncer) HandleBlockBodiesMsg(peerId models.P2PID, request []*models.Body) {
	if request == nil || len(request) == 0 {
		return
	}
	for _, body := range request {
		s.bodyHandle(body)
	}
}

func (s *syncer) bodyHandle(body *models.Body) {
	block := s.queues[body.Height]
	if block == nil {
		s.log.Error("bodyHandle is error", "blockHeight", body.Height)
		return
	}
	block.SetTransactions(body.Txs)
	block.syncing = false
	// 发送chan
	s.blockCompletedCh <- struct{}{}
}

func (s *syncer) headerHandle(peerId models.P2PID, header *models.Header) {
	remoteHeight := header.Height
	remoteHash := header.Hash()
	tempBlock := s.queues[remoteHeight]
	if tempBlock != nil {
		// 本地已经存有height
		if tempBlock.Hash() != remoteHash {
			// TODO 两边的hash不一致
		}
	}
	peerBlock := NewPeerBlock(header)
	s.queues[remoteHeight] = peerBlock
}
