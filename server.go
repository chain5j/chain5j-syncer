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

const (
	softResponseLimit = 2 * 1024 * 1024 // Target maximum size of returned blocks, headers or node data.
	estHeaderRlpSize  = 500             // Approximate size of an RLP encoded block header

)

const (
	StatusMsg          = 0x00
	NewBlockHashesMsg  = 0x01
	TxMsg              = 0x02
	GetBlockHeadersMsg = 0x03
	BlockHeadersMsg    = 0x04
	GetBlockBodiesMsg  = 0x05
	BlockBodiesMsg     = 0x06
	NewBlockMsg        = 0x07
	GetNodeDataMsg     = 0x0d
	NodeDataMsg        = 0x0e
	GetReceiptsMsg     = 0x0f
	ReceiptsMsg        = 0x10
)

var (
	MaxHashFetch    = 512 // Amount of hashes to be fetched per retrieval request
	MaxBlockFetch   = 128 // Amount of blocks to be fetched per retrieval request
	MaxHeaderFetch  = 192 // Amount of block headers to be fetched per retrieval request
	MaxSkeletonSize = 128 // Number of header fetches to need for a skeleton assembly
	MaxBodyFetch    = 128 // Amount of block bodies to be fetched per retrieval request
	MaxReceiptFetch = 256 // Amount of transaction receipts to allow fetching per request
	MaxStateFetch   = 384 // Amount of node state values to allow fetching per request

)

// 查询BlockHeader进行发送
func (s *syncer) SendBlockHeaders(peerId models.P2PID, query ext.GetBlockHeadersData) {
	hashMode := query.Origin.Hash != (types.Hash{})
	first := true
	maxNonCanonical := uint64(100)

	var (
		bytes   types.StorageSize
		headers []*models.Header
		unknown bool
	)

	for !unknown && len(headers) < int(query.Amount) && bytes < softResponseLimit && len(headers) < MaxHeaderFetch {
		// Retrieve the next header satisfying the query
		var origin *models.Header
		if hashMode {
			if first {
				first = false
				origin = s.blockRW.GetHeaderByHash(query.Origin.Hash)
				if origin != nil {
					query.Origin.Number = origin.Height
				}
			} else {
				origin = s.blockRW.GetHeader(query.Origin.Hash, query.Origin.Number)
			}
		} else {
			origin = s.blockRW.GetHeaderByNumber(query.Origin.Number)
		}
		if origin == nil {
			break
		}
		headers = append(headers, origin)
		bytes += estHeaderRlpSize

		// Advance to the next header of the query
		switch {
		case hashMode && query.Reverse:
			// Hash based traversal towards the genesis block
			// ancestor := query.Skip + 1
			ancestor := uint64(1)
			if ancestor == 0 {
				unknown = true
			} else {
				query.Origin.Hash, query.Origin.Number = s.blockRW.GetAncestor(query.Origin.Hash, query.Origin.Number, ancestor, &maxNonCanonical)
				unknown = (query.Origin.Hash == types.Hash{})
			}
		case hashMode && !query.Reverse:
			// Hash based traversal towards the leaf block
			var (
				current = origin.Height
				// next    = current + query.Skip + 1
				next = current + 1
			)
			if next <= current {
				// infos, _ := json.MarshalIndent(s.p2p.Info(), "", "  ")
				infos := s.p2p.Id()
				// s.log.Warn("GetBlockHeaders skip overflow attack", "current", current, "skip", query.Skip, "next", next, "attacker", infos)
				s.log.Warn("GetBlockHeaders skip overflow attack", "current", current, "next", next, "attacker", infos)
				unknown = true
			} else {
				if header := s.blockRW.GetHeaderByNumber(next); header != nil {
					nextHash := header.Hash()
					// expOldHash, _ := s.blockRW.GetAncestor(nextHash, next, query.Skip+1, &maxNonCanonical)
					expOldHash, _ := s.blockRW.GetAncestor(nextHash, next, 1, &maxNonCanonical)
					if expOldHash == query.Origin.Hash {
						query.Origin.Hash, query.Origin.Number = nextHash, next
					} else {
						unknown = true
					}
				} else {
					unknown = true
				}
			}
		case query.Reverse:
			// Number based traversal towards the genesis block
			// if query.Origin.Number >= query.Skip+1 {
			//	query.Origin.Number -= query.Skip + 1
			// } else {
			//	unknown = true
			// }
			if query.Origin.Number >= 1 {
				query.Origin.Number -= 1
			} else {
				unknown = true
			}

		case !query.Reverse:
			// Number based traversal towards the leaf block
			// query.Origin.Number += query.Skip + 1
			query.Origin.Number += 1
		}
	}
	toBytes, err := codec.Coder().Encode(headers)
	if err != nil {
		s.log.Error("headers codec.Encode err", "err", err)
		return
	}

	go s.p2p.Send(peerId, &models.P2PMessage{
		Type: BlockHeadersMsg,
		Peer: "",
		Data: toBytes,
	})
}

// ====================body==============
func (s *syncer) SendBlockBodies(peerId models.P2PID, hashes []types.Hash) {
	if hashes == nil || len(hashes) == 0 {
		return
	}

	var body *models.Body

	var bodies []*models.Body
	for _, blockHash := range hashes {
		body = s.blockRW.GetBody(blockHash)
		bodies = append(bodies, body)
	}

	toBytes, err := codec.Coder().Encode(bodies)
	if err != nil {
		s.log.Error("bodies codec.Encode err", "err", err)
		return
	}
	go s.p2p.Send(peerId, &models.P2PMessage{
		Type: BlockBodiesMsg,
		Peer: "",
		Data: toBytes,
	})
}

func (s *syncer) listen() error {
	// =========接收请求=========
	// 监听获取header
	getBlockHeadersMsgCh := make(chan *models.P2PMessage, 1)
	getBlockHeadersMsgSub := s.p2p.SubscribeMsg(GetBlockHeadersMsg, getBlockHeadersMsgCh)

	getBlockBodiesMsgCh := make(chan *models.P2PMessage, 1)
	getBlockBodiesMsgSub := s.p2p.SubscribeMsg(GetBlockBodiesMsg, getBlockBodiesMsgCh)

	// =========接收响应=========
	respondBlockHeadersMsgCh := make(chan *models.P2PMessage, 1)
	respondBlockHeadersMsgSub := s.p2p.SubscribeMsg(BlockHeadersMsg, respondBlockHeadersMsgCh)

	respondBlockBodiesMsgCh := make(chan *models.P2PMessage, 1)
	respondBlockBodiesMsgSub := s.p2p.SubscribeMsg(BlockBodiesMsg, respondBlockBodiesMsgCh)

	for {
		select {
		// 接收到请求
		case blockHeadersMsg := <-getBlockHeadersMsgCh:
			data := blockHeadersMsg.Data
			var query ext.GetBlockHeadersData
			if err := codec.Coder().Decode(data, &query); err != nil {
				s.log.Error("getBlockHeadersData decode err", "msg", blockHeadersMsg, "err", err)
				break
			}
			s.SendBlockHeaders(blockHeadersMsg.Peer, query)
		case err := <-getBlockHeadersMsgSub.Err():
			s.log.Error("getBlockHeadersMsgSub err", "err", err)
			break

		case ch := <-getBlockBodiesMsgCh:
			data := ch.Data
			var hashes []types.Hash
			if err := codec.Coder().Decode(data, &hashes); err != nil {
				s.log.Error("getBlockBodiesMsgCh decode err", "msg", ch, "err", err)
				break
			}
			s.SendBlockBodies(ch.Peer, hashes)
		case err := <-getBlockBodiesMsgSub.Err():
			s.log.Error("getBlockBodiesMsgSub err", "err", err)
			break

		// 接收响应
		case blockHeadersMsg := <-respondBlockHeadersMsgCh:
			var headers []*models.Header
			if err := codec.Coder().Decode(blockHeadersMsg.Data, &headers); err != nil {
				s.log.Error("blockHeadersMsg decode err", "msg", blockHeadersMsg, "err", err)
				break
			}
			s.HandleBlockHeadersMsg(blockHeadersMsg.Peer, headers)
		case err := <-respondBlockHeadersMsgSub.Err():
			s.log.Error("respondBlockHeadersMsgSub err", "err", err)
			break

		case blockBodiesMsg := <-respondBlockBodiesMsgCh:
			var request []*models.Body
			if err := codec.Coder().Decode(blockBodiesMsg.Data, &request); err != nil {
				s.log.Error("blockBodiesMsg decode err", "msg", blockBodiesMsg, "err", err)
				break
			}
			s.HandleBlockBodiesMsg(blockBodiesMsg.Peer, request)
		case err := <-respondBlockBodiesMsgSub.Err():
			s.log.Error("respondBlockBodiesMsgSub err", "err", err)
			break

		}
	}
}
