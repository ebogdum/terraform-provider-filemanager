// SPDX-License-Identifier: MIT

package ssh

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// PoolConfig configures the connection pool.
type PoolConfig struct {
	// MaxConnections is the maximum number of connections to maintain.
	MaxConnections int

	// IdleTimeout is how long an idle connection can remain in the pool.
	IdleTimeout time.Duration

	// HealthCheckInterval is how often to check connection health.
	HealthCheckInterval time.Duration

	// MaxRetries is the maximum number of connection attempts.
	MaxRetries int

	// RetryDelay is the initial delay between connection attempts.
	RetryDelay time.Duration
}

// DefaultPoolConfig returns a sensible default pool configuration.
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxConnections:      5,
		IdleTimeout:         5 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
		MaxRetries:          3,
		RetryDelay:          1 * time.Second,
	}
}

// PooledConnection represents a connection in the pool.
type PooledConnection struct {
	SSHClient  *ssh.Client
	SFTPClient *sftp.Client
	CreatedAt  time.Time
	LastUsedAt time.Time
	InUse      bool
}

// ConnectionPool manages a pool of SSH/SFTP connections.
type ConnectionPool struct {
	config      PoolConfig
	connections []*PooledConnection
	mu          sync.Mutex
	closed      bool

	// Connection factory
	dial    func() (*ssh.Client, error)
	sshConf *ssh.ClientConfig
	address string

	// Health check goroutine lifecycle
	healthCheckStop chan struct{}
	healthCheckDone chan struct{}
}

// NewConnectionPool creates a new connection pool.
func NewConnectionPool(config PoolConfig) *ConnectionPool {
	return &ConnectionPool{
		config:      config,
		connections: make([]*PooledConnection, 0, config.MaxConnections),
	}
}

// Configure sets up the connection factory.
func (p *ConnectionPool) Configure(address string, config *ssh.ClientConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.address = address
	p.sshConf = config
	p.dial = func() (*ssh.Client, error) {
		return ssh.Dial("tcp", address, config)
	}
}

// Get retrieves an available connection from the pool, or creates a new one.
func (p *ConnectionPool) Get(ctx context.Context) (*PooledConnection, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, fmt.Errorf("connection pool is closed")
	}

	// Find first idle connection that passes basic health check (no network call under lock)
	for _, conn := range p.connections {
		if !conn.InUse && p.isHealthyNoLock(conn) {
			conn.InUse = true
			conn.LastUsedAt = time.Now()
			p.mu.Unlock()
			return conn, nil
		}
	}

	// Create a new connection if under limit
	if len(p.connections) < p.config.MaxConnections {
		conn, err := p.createConnection(ctx)
		if nil != err {
			p.mu.Unlock()
			return nil, err
		}
		p.connections = append(p.connections, conn)
		p.mu.Unlock()
		return conn, nil
	}
	p.mu.Unlock()

	return nil, fmt.Errorf("connection pool exhausted")
}

// Put returns a connection to the pool.
func (p *ConnectionPool) Put(conn *PooledConnection) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if nil == conn {
		return
	}

	conn.InUse = false
	conn.LastUsedAt = time.Now()
}

// Release releases a connection and removes it from the pool (e.g., on error).
func (p *ConnectionPool) Release(conn *PooledConnection) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if nil == conn {
		return
	}

	// Close the connection
	if nil != conn.SFTPClient {
		conn.SFTPClient.Close()
	}
	if nil != conn.SSHClient {
		conn.SSHClient.Close()
	}

	// Remove from pool
	for i, c := range p.connections {
		if c == conn {
			p.connections = append(p.connections[:i], p.connections[i+1:]...)
			break
		}
	}
}

// Close closes all connections in the pool and stops the health check goroutine.
func (p *ConnectionPool) Close() error {
	// Stop health check first (outside of lock to avoid deadlock)
	p.StopHealthCheck()

	p.mu.Lock()
	defer p.mu.Unlock()

	p.closed = true

	var errs []error
	for _, conn := range p.connections {
		if nil != conn.SFTPClient {
			if err := conn.SFTPClient.Close(); nil != err {
				errs = append(errs, err)
			}
		}
		if nil != conn.SSHClient {
			if err := conn.SSHClient.Close(); nil != err {
				errs = append(errs, err)
			}
		}
	}

	p.connections = nil

	if len(errs) > 0 {
		return fmt.Errorf("errors closing pool connections: %v", errs)
	}
	return nil
}

// createConnection creates a new SSH/SFTP connection with exponential backoff and jitter.
func (p *ConnectionPool) createConnection(ctx context.Context) (*PooledConnection, error) {
	if nil == p.dial {
		return nil, fmt.Errorf("connection pool not configured")
	}

	var sshClient *ssh.Client
	var err error

	// Retry with exponential backoff and jitter
	delay := p.config.RetryDelay
	const maxDelay = 30 * time.Second

	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		if attempt > 0 {
			jitter := time.Duration(rand.Int64N(int64(delay) / 2))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay + jitter):
			}
			delay = delay * 2
			if delay > maxDelay {
				delay = maxDelay
			}
		}

		sshClient, err = p.dial()
		if nil == err {
			break
		}
	}

	if nil != err {
		return nil, fmt.Errorf("failed to connect after %d attempts: %w", p.config.MaxRetries+1, err)
	}

	// Create SFTP client
	sftpClient, err := sftp.NewClient(sshClient)
	if nil != err {
		sshClient.Close()
		return nil, fmt.Errorf("failed to create SFTP client: %w", err)
	}

	now := time.Now()
	return &PooledConnection{
		SSHClient:  sshClient,
		SFTPClient: sftpClient,
		CreatedAt:  now,
		LastUsedAt: now,
		InUse:      true,
	}, nil
}

// Cleanup removes idle and unhealthy connections from the pool.
// Connections are closed outside the lock to avoid blocking pool operations on network timeouts.
func (p *ConnectionPool) Cleanup() {
	p.mu.Lock()
	var healthy []*PooledConnection
	var toClose []*PooledConnection
	for _, conn := range p.connections {
		if conn.InUse || p.isHealthyNoLock(conn) {
			healthy = append(healthy, conn)
		} else {
			toClose = append(toClose, conn)
		}
	}
	p.connections = healthy
	p.mu.Unlock()

	// Close outside the lock
	for _, conn := range toClose {
		if nil != conn.SFTPClient {
			conn.SFTPClient.Close()
		}
		if nil != conn.SSHClient {
			conn.SSHClient.Close()
		}
	}
}

// isHealthyNoLock checks health without acquiring the lock (for internal use).
func (p *ConnectionPool) isHealthyNoLock(conn *PooledConnection) bool {
	if nil == conn || nil == conn.SSHClient || nil == conn.SFTPClient {
		return false
	}

	// Check if connection has been idle too long
	if time.Since(conn.LastUsedAt) > p.config.IdleTimeout {
		return false
	}

	return true
}

// Stats returns pool statistics.
type PoolStats struct {
	Total     int
	InUse     int
	Available int
	Idle      int
}

// Stats returns current pool statistics.
func (p *ConnectionPool) Stats() PoolStats {
	p.mu.Lock()
	defer p.mu.Unlock()

	stats := PoolStats{
		Total: len(p.connections),
	}

	for _, conn := range p.connections {
		if conn.InUse {
			stats.InUse++
		} else {
			stats.Available++
			if time.Since(conn.LastUsedAt) > p.config.IdleTimeout/2 {
				stats.Idle++
			}
		}
	}

	return stats
}

// StartHealthCheck starts a background goroutine that periodically cleans up the pool.
// The goroutine stops when ctx is cancelled, Stop() is called, or the pool is closed.
func (p *ConnectionPool) StartHealthCheck(ctx context.Context) {
	if p.config.HealthCheckInterval <= 0 {
		return
	}

	p.mu.Lock()
	// Don't start if pool is already closed
	if p.closed {
		p.mu.Unlock()
		return
	}
	// Don't start a second health check
	if nil != p.healthCheckStop {
		p.mu.Unlock()
		return
	}
	p.healthCheckStop = make(chan struct{})
	p.healthCheckDone = make(chan struct{})
	p.mu.Unlock()

	go func() {
		ticker := time.NewTicker(p.config.HealthCheckInterval)
		defer ticker.Stop()
		defer close(p.healthCheckDone)

		for {
			select {
			case <-ctx.Done():
				return
			case <-p.healthCheckStop:
				return
			case <-ticker.C:
				p.mu.Lock()
				if p.closed {
					p.mu.Unlock()
					return
				}
				p.mu.Unlock()
				p.Cleanup()
			}
		}
	}()
}

// StopHealthCheck stops the health check goroutine and waits for it to finish.
func (p *ConnectionPool) StopHealthCheck() {
	p.mu.Lock()
	if nil == p.healthCheckStop {
		p.mu.Unlock()
		return
	}
	stop := p.healthCheckStop
	done := p.healthCheckDone
	p.mu.Unlock()

	close(stop)
	<-done

	p.mu.Lock()
	p.healthCheckStop = nil
	p.healthCheckDone = nil
	p.mu.Unlock()
}

// WarmUp pre-creates a number of connections.
func (p *ConnectionPool) WarmUp(ctx context.Context, count int) error {
	if count > p.config.MaxConnections {
		count = p.config.MaxConnections
	}

	for i := 0; i < count; i++ {
		conn, err := p.Get(ctx)
		if nil != err {
			return fmt.Errorf("failed to warm up connection %d: %w", i+1, err)
		}
		p.Put(conn)
	}

	return nil
}

// Size returns the current number of connections in the pool.
func (p *ConnectionPool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.connections)
}
