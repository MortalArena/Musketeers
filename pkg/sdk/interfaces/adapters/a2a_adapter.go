package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/MortalArena/Musketeers/pkg/node"
	"github.com/MortalArena/Musketeers/pkg/sdk/interfaces"
	"github.com/libp2p/go-libp2p/core/peer"
)

type A2AAdapter struct {
	node   *node.Node
	msgCh  chan *interfaces.A2AMessage
	once   sync.Once
}

func NewA2AAdapter(n *node.Node) *A2AAdapter {
	return &A2AAdapter{node: n}
}

func (a *A2AAdapter) Send(ctx context.Context, target string, msg *interfaces.A2AMessage) error {
	pid, err := peer.Decode(target)
	if err != nil {
		return err
	}
	var input interface{}
	if len(msg.Payload) > 0 {
		json.Unmarshal(msg.Payload, &input)
	}
	_, err = a.node.SendACPTask(ctx, pid, msg.Target, msg.Type, input)
	return err
}

func (a *A2AAdapter) Receive(ctx context.Context) (*interfaces.A2AMessage, error) {
	a.once.Do(func() {
		a.msgCh = make(chan *interfaces.A2AMessage, 100)
		a.node.RegisterACPTask("task", func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
			msg := &interfaces.A2AMessage{
				Payload: input,
			}
			select {
			case a.msgCh <- msg:
			default:
			}
			return nil, fmt.Errorf("A2A Receive: reply not supported via polling, use RegisterHandler")
		})
	})
	select {
	case msg := <-a.msgCh:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (a *A2AAdapter) RegisterHandler(handler func(ctx context.Context, msg *interfaces.A2AMessage) (*interfaces.A2AMessage, error)) error {
	a.node.RegisterACPTask("task", func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
		in := &interfaces.A2AMessage{
			Payload: input,
		}
		out, err := handler(ctx, in)
		if err != nil {
			return nil, err
		}
		return out.Payload, nil
	})
	return nil
}

var _ interfaces.A2AInterface = (*A2AAdapter)(nil)
