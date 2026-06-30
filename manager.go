package paykit

import (
	"context"
	"fmt"
	"net/http"
	"sync"
)

type Manager struct {
	mu        sync.RWMutex
	providers map[string]Provider
	defaultP  string
}

func NewManager() *Manager {
	return &Manager{providers: make(map[string]Provider)}
}

func (m *Manager) Register(p Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.providers[p.Name()]; !ok && m.defaultP == "" {
		m.defaultP = p.Name()
	}
	m.providers[p.Name()] = p
}

func (m *Manager) SetDefault(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.providers[name]; !ok {
		return fmt.Errorf("paykit: provider %q not registered", name)
	}
	m.defaultP = name
	return nil
}

func (m *Manager) Get(name string) (Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.providers[name]
	if !ok {
		return nil, fmt.Errorf("paykit: provider %q not registered", name)
	}
	return p, nil
}

func (m *Manager) CreatePayment(ctx context.Context, req CreatePaymentRequest) (*Payment, error) {
	m.mu.RLock()
	p := m.providers[m.defaultP]
	m.mu.RUnlock()
	if p == nil {
		return nil, fmt.Errorf("paykit: no default provider")
	}
	return p.CreatePayment(ctx, req)
}

func (m *Manager) HandleWebhook(onEvent func(*WebhookEvent)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("provider") // requires Go 1.22+ ServeMux
		p, err := m.Get(name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		ev, err := p.ParseWebhook(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if onEvent != nil {
			onEvent(ev)
		}
		status, body := p.WebhookAck(ev)
		if status == 0 {
			status = http.StatusOK
		}
		w.WriteHeader(status)
		if len(body) > 0 {
			_, _ = w.Write(body)
		}
	}
}
