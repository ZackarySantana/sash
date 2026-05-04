package api

import (
	"context"
	"sync"
)

type API struct {
	mu sync.Mutex
	n  int
}

func (a *API) Get(ctx context.Context) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.n, nil
}

func (a *API) Add(ctx context.Context, delta int) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.n += delta
	return a.n, nil
}
