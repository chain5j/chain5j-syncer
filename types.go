// Package syncer
//
// @author: xwc1125
package syncer

import (
	"github.com/chain5j/chain5j-protocol/models"
)

// ==============peerBlock=============
type peerBlock struct {
	*models.Block
	syncing bool //是否在同步中
}

func NewPeerBlock(header *models.Header) *peerBlock {
	return &peerBlock{
		Block:   models.NewBlock(header, nil, nil),
		syncing: true,
	}
}

//type getBodiesData struct {
//	blockHeight uint64
//}

// ==============body=============
//// blockBody represents the data content of a single block.
//type blockBody struct {
//	Transactions []models.Transaction // Transactions contained within a block
//	Uncles       []*models.Header     // Uncles contained within a block
//}
//
//// blockBodiesData is the network packet for block content distribution.
//type blockBodiesData []*blockBody
