package global

import (
	"context"
	"sync"
)

var DefaultStreamRegisterer = new(StreamRegisterer)

// Stream describes the interface to be implemented for accessing
// the data stream generated by the enabled agent watchers.
type Stream interface {
	// Start receives a context and a wait group passed by the main routine
	// for gracefully exit and the channel to read from.
	Start(ctx context.Context, wg *sync.WaitGroup, ch chan interface{})
}

type StreamRegisterer struct {
	streams []Stream
}

func newRegisterer() *StreamRegisterer {
	return &StreamRegisterer{}
}

func (r *StreamRegisterer) Register(w ...Stream) error {
	r.streams = append(r.streams, w...)

	return nil
}

func (r *StreamRegisterer) Start(ctx context.Context, wg *sync.WaitGroup, ch chan interface{}) error {
	for _, s := range r.streams {
		s.Start(ctx, wg, ch)
	}

	return nil
}