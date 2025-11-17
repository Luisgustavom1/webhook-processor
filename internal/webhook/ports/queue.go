package ports

import (
	"context"
)

type QueuePortPublishOpts struct {
	Delay int
}

type QueuePort interface {
	Publish(ctx context.Context, msg []byte, opts QueuePortPublishOpts) error
}
