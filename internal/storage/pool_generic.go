package storage

import (
	"sync"
	"time"
)

type GenericConnectionPool struct {
	connections map[string]*PooledGenericConnection
	mu          sync.RWMutex
	ttl         time.Duration
	backend     StorageBackend
}

type PooledGenericConnection struct {
	connection Connection
	lastUsed   time.Time
	username   string
}

func NewGenericConnectionPool(backend StorageBackend, ttl time.Duration) *GenericConnectionPool {
	pool := &GenericConnectionPool{
		connections: make(map[string]*PooledGenericConnection),
		ttl:         ttl,
		backend:     backend,
	}

	go pool.cleanup()

	return pool
}

func (p *GenericConnectionPool) GetConnection(username, password string) (Connection, error) {
	p.mu.RLock()
	conn, exists := p.connections[username]
	p.mu.RUnlock()

	if exists && time.Since(conn.lastUsed) < p.ttl {
		conn.lastUsed = time.Now()
		return conn.connection, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if conn, exists := p.connections[username]; exists && time.Since(conn.lastUsed) < p.ttl {
		conn.lastUsed = time.Now()
		return conn.connection, nil
	}

	if exists {
		p.backend.Close(conn.connection)
	}

	connection, err := p.backend.Connect(username, password)
	if err != nil {
		return nil, err
	}

	p.connections[username] = &PooledGenericConnection{
		connection: connection,
		lastUsed:   time.Now(),
		username:   username,
	}

	return connection, nil
}

func (p *GenericConnectionPool) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()

		now := time.Now()
		for username, conn := range p.connections {
			if now.Sub(conn.lastUsed) > p.ttl {
				p.backend.Close(conn.connection)
				delete(p.connections, username)
			}
		}

		p.mu.Unlock()
	}
}

func (p *GenericConnectionPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, conn := range p.connections {
		p.backend.Close(conn.connection)
	}

	p.connections = make(map[string]*PooledGenericConnection)
}
