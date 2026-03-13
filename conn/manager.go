package conn

import (
	"sync"
)

type Manager struct {
	mu    sync.RWMutex
	conns map[int64]*Client
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
