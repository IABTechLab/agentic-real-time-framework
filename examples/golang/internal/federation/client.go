// Copyright (c) 2025 Index Exchange Inc.
//
// This file is part of the Agentic RTB Framework reference implementation.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package federation

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	pb "github.com/iabtechlab/agentic-rtb-framework/pkg/pb/artf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// Client wraps a GRPC connection to a federated endpoint
type Client struct {
	config    EndpointConfig
	conn      *grpc.ClientConn
	rtbClient pb.RTBExtensionPointClient
	mu        sync.RWMutex
	healthy   bool
	lastError error
	lastCheck time.Time
}

// ClientPool manages connections to multiple federated endpoints
type ClientPool struct {
	config  *Config
	clients map[string]*Client
	mu      sync.RWMutex
}

// NewClientPool creates a new client pool from configuration
func NewClientPool(config *Config) (*ClientPool, error) {
	pool := &ClientPool{
		config:  config,
		clients: make(map[string]*Client),
	}

	// Initialize clients for all enabled endpoints
	for _, ep := range config.GetEnabledEndpoints() {
		client, err := NewClient(ep, config.Defaults)
		if err != nil {
			log.Printf("[Federation] Failed to create client for endpoint '%s': %v", ep.Name, err)
			continue
		}
		pool.clients[ep.Name] = client
		log.Printf("[Federation] Initialized client for endpoint '%s' at %s", ep.Name, ep.Address)
	}

	return pool, nil
}

// NewClient creates a new client for a single endpoint
func NewClient(config EndpointConfig, defaults *EndpointDefaults) (*Client, error) {
	opts := []grpc.DialOption{
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(10 * 1024 * 1024), // 10MB
			grpc.MaxCallSendMsgSize(10 * 1024 * 1024),
		),
	}

	// Configure TLS
	tlsConfig := config.TLS
	if tlsConfig == nil && defaults != nil {
		tlsConfig = defaults.TLS
	}

	if tlsConfig != nil && tlsConfig.Enabled {
		creds, err := buildTLSCredentials(tlsConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to configure TLS: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Connect
	conn, err := grpc.NewClient(config.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", config.Address, err)
	}

	client := &Client{
		config:    config,
		conn:      conn,
		rtbClient: pb.NewRTBExtensionPointClient(conn),
		healthy:   true,
	}

	return client, nil
}

// buildTLSCredentials creates TLS credentials from config
func buildTLSCredentials(config *TLSConfig) (credentials.TransportCredentials, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.Insecure,
	}

	if config.ServerName != "" {
		tlsConfig.ServerName = config.ServerName
	}

	// Load CA certificate if provided
	if config.CAFile != "" {
		caCert, err := os.ReadFile(config.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA file: %w", err)
		}
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = certPool
	}

	// Load client certificate if provided
	if config.CertFile != "" && config.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return credentials.NewTLS(tlsConfig), nil
}

// GetMutations calls the RTBExtensionPoint GRPC service
func (c *Client) GetMutations(ctx context.Context, req *pb.RTBRequest) (*pb.RTBResponse, error) {
	c.mu.RLock()
	if !c.healthy {
		c.mu.RUnlock()
		return nil, fmt.Errorf("endpoint '%s' is unhealthy: %v", c.config.Name, c.lastError)
	}
	c.mu.RUnlock()

	// Apply timeout
	timeout := time.Duration(c.config.GetTimeoutMs(nil)) * time.Millisecond
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if c.rtbClient == nil {
		return nil, fmt.Errorf("RTBExtensionPoint client not initialized")
	}

	resp, err := c.rtbClient.GetMutations(ctx, req)
	if err != nil {
		c.markUnhealthy(err)
		return nil, err
	}
	return resp, nil
}

// markUnhealthy marks the client as unhealthy
func (c *Client) markUnhealthy(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.healthy = false
	c.lastError = err
	c.lastCheck = time.Now()
	log.Printf("[Federation] Endpoint '%s' marked unhealthy: %v", c.config.Name, err)
}

// markHealthy marks the client as healthy
func (c *Client) markHealthy() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.healthy {
		log.Printf("[Federation] Endpoint '%s' recovered", c.config.Name)
	}
	c.healthy = true
	c.lastError = nil
	c.lastCheck = time.Now()
}

// IsHealthy returns whether the client is healthy
func (c *Client) IsHealthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.healthy
}

// Config returns the endpoint configuration
func (c *Client) Config() EndpointConfig {
	return c.config
}

// Close closes the client connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetClient returns a client by endpoint name
func (p *ClientPool) GetClient(name string) *Client {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.clients[name]
}

// GetClients returns all clients
func (p *ClientPool) GetClients() []*Client {
	p.mu.RLock()
	defer p.mu.RUnlock()
	clients := make([]*Client, 0, len(p.clients))
	for _, c := range p.clients {
		clients = append(clients, c)
	}
	return clients
}

// GetClientsByIntent returns clients that accept the given intent
func (p *ClientPool) GetClientsByIntent(intent string) []*Client {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var clients []*Client
	for _, c := range p.clients {
		if c.config.HasIntent(intent) && c.IsHealthy() {
			clients = append(clients, c)
		}
	}
	return clients
}

// GetHealthyClients returns all healthy clients
func (p *ClientPool) GetHealthyClients() []*Client {
	p.mu.RLock()
	defer p.mu.RUnlock()
	var clients []*Client
	for _, c := range p.clients {
		if c.IsHealthy() {
			clients = append(clients, c)
		}
	}
	return clients
}

// Close closes all client connections
func (p *ClientPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	var lastErr error
	for name, c := range p.clients {
		if err := c.Close(); err != nil {
			log.Printf("[Federation] Error closing client '%s': %v", name, err)
			lastErr = err
		}
	}
	return lastErr
}
