package conn

import (
	"sync"
)

type Manager struct {
	mu    sync.RWMutex
	conns map[int64]*Client
}

// ConnItem is a snapshot item for safe iteration outside lock.
type ConnItem struct {
	UID    int64
	Client *Client
}

func NewManager() *Manager {
	return &Manager{
		conns: make(map[int64]*Client),
	}
}

func (m *Manager) Add(uid int64, c *Client) {
	m.mu.Lock()
	m.conns[uid] = c
	m.mu.Unlock()
}

func (m *Manager) Get(uid int64) (*Client, bool) {
	m.mu.RLock()
	c, ok := m.conns[uid]
	m.mu.RUnlock()
	return c, ok
}

func (m *Manager) Remove(uid int64) {
	m.mu.Lock()
	delete(m.conns, uid)
	m.mu.Unlock()
}

func (m *Manager) Range(fn func(uid int64, cli *Client)) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for uid, cli := range m.conns {
		fn(uid, cli)
	}
}

// Snapshot returns a copy of current connections for iteration without holding the manager lock.
func (m *Manager) Snapshot() []ConnItem {
	m.mu.RLock()
	items := make([]ConnItem, 0, len(m.conns))
	for uid, cli := range m.conns {
		items = append(items, ConnItem{UID: uid, Client: cli})
	}
	m.mu.RUnlock()
	return items
}
