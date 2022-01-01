// Package syncer
//
// @author: xwc1125
package syncer

import (
	"fmt"
	"github.com/chain5j/chain5j-protocol/protocol"
)

type option func(f *syncer) error

func apply(f *syncer, opts ...option) error {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(f); err != nil {
			return fmt.Errorf("option apply err:%v", err)
		}
	}
	return nil
}

func WithP2PService(p2pService protocol.P2PService) option {
	return func(f *syncer) error {
		f.p2p = p2pService
		return nil
	}
}
func WithApps(apps protocol.Apps) option {
	return func(f *syncer) error {
		f.apps = apps
		return nil
	}
}
func WithBlockRW(blockRW protocol.BlockReadWriter) option {
	return func(f *syncer) error {
		f.blockRW = blockRW
		return nil
	}
}

func WithHandshake(handshake protocol.Handshake) option {
	return func(f *syncer) error {
		f.handshake = handshake
		return nil
	}
}
