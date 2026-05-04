package api

import (
	"context"
	"time"

	"github.com/zackarysantana/sash/src/sashsse"
)

type API struct {
	events *sashsse.Broker
}

func New(events *sashsse.Broker) *API {
	return &API{events: events}
}

func (*API) Message(ctx context.Context) (string, error) {
	return "Hello from Go · Solid · Vite · Tailwind", nil
}

func (a *API) WaitFiveSeconds(ctx context.Context) error {
	for i := 1; i <= 5; i++ {
		_ = a.events.PublishJSON("tick", map[string]int{"sec": i})
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return nil
}
