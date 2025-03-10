package pool

import (
	"context"
	"sync"
	"time"

	"github.com/tonkeeper/tongo/liteclient"
	"github.com/tonkeeper/tongo/ton"
)

type connection struct {
	id     int
	client *liteclient.Client

	// masterHeadUpdatedCh is used to send a notification when a known master head is changed.
	masterHeadUpdatedCh chan masterHeadUpdated

	mu sync.RWMutex
	// masterHead is the latest known masterchain head.
	masterHead ton.BlockIDExt
}

type masterHeadUpdated struct {
	Head ton.BlockIDExt
	Conn *connection
}

func (c *connection) Run(ctx context.Context) {
	for {
		var head ton.BlockIDExt
		for {
			res, err := c.client.LiteServerGetMasterchainInfo(ctx)
			if err != nil {
				// TODO: log error
				time.Sleep(1000 * time.Millisecond)
				continue
			}
			head = res.Last.ToBlockIdExt()
			break
		}
		c.SetMasterHead(head)
		for {
			if err := c.client.WaitMasterchainSeqno(ctx, head.Seqno+1, 15_000); err != nil {
				// TODO: log error
				time.Sleep(1000 * time.Millisecond)
				// we want to request seqno again with LiteServerGetMasterchainInfo
				// to avoid situation when this server has been offline for too long,
				// and it doesn't contain a block with the latest known seqno anymore.
				break
			}
			if ctx.Err() != nil {
				return
			}
			res, err := c.client.LiteServerGetMasterchainInfo(ctx)
			if err != nil {
				// TODO: log error
				time.Sleep(1000 * time.Millisecond)
				// we want to request seqno again with LiteServerGetMasterchainInfo
				// to avoid situation when this server has been offline for too long,
				// and it doesn't contain a block with the latest known seqno anymore.
				break
			}
			if ctx.Err() != nil {
				return
			}
			head = res.Last.ToBlockIdExt()
			c.SetMasterHead(head)
		}
	}
}

// IsOK returns true if there is no problems with the underlying liteclient and its connection to a lite server.
func (c *connection) IsOK() bool {
	return c.client.IsOK()
}

func (c *connection) ID() int {
	return c.id
}

func (c *connection) Client() *liteclient.Client {
	return c.client
}

func (c *connection) MasterHead() ton.BlockIDExt {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.masterHead
}

func (c *connection) SetMasterHead(head ton.BlockIDExt) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if head.Seqno > c.masterHead.Seqno {
		c.masterHead = head
		c.masterHeadUpdatedCh <- masterHeadUpdated{
			Head: head,
			Conn: c,
		}
	}
}
