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
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	pb "github.com/iabtechlab/agentic-rtb-framework/pkg/pb/artf"
)

// Manager coordinates federated GRPC calls across multiple endpoints
type Manager struct {
	pool   *ClientPool
	config *Config
}

// FederatedResult contains the result from a single federated endpoint
type FederatedResult struct {
	EndpointName string         `json:"endpoint_name"`
	Success      bool           `json:"success"`
	Mutations    []*pb.Mutation `json:"mutations,omitempty"`
	Error        string         `json:"error,omitempty"`
	LatencyMs    int64          `json:"latency_ms"`
}

// FederatedResponse contains aggregated results from all endpoints
type FederatedResponse struct {
	ID              string            `json:"id"`
	Mutations       []*pb.Mutation    `json:"mutations"`
	EndpointResults []FederatedResult `json:"endpoint_results"`
	TotalLatencyMs  int64             `json:"total_latency_ms"`
	Metadata        *pb.Metadata      `json:"metadata,omitempty"`
}

// EndpointInfo provides information about a federated endpoint
type EndpointInfo struct {
	Name              string   `json:"name"`
	Address           string   `json:"address"`
	Description       string   `json:"description,omitempty"`
	Service           string   `json:"service"`
	ApplicableIntents []string `json:"applicable_intents"`
	Priority          int      `json:"priority"`
	Enabled           bool     `json:"enabled"`
	Healthy           bool     `json:"healthy"`
	TimeoutMs         int      `json:"timeout_ms"`
}

// NewManager creates a new federation manager
func NewManager(config *Config) (*Manager, error) {
	pool, err := NewClientPool(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create client pool: %w", err)
	}

	return &Manager{
		pool:   pool,
		config: config,
	}, nil
}

// NewManagerFromFile loads config from a file and creates a manager
func NewManagerFromFile(configPath string) (*Manager, error) {
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return NewManager(config)
}

// GetMutations calls all applicable federated endpoints and aggregates results
func (m *Manager) GetMutations(ctx context.Context, req *pb.RTBRequest, acceptableIntents []string) (*FederatedResponse, error) {
	startTime := time.Now()

	// Get endpoints that match the acceptable intents
	var clients []*Client
	if len(acceptableIntents) == 0 {
		clients = m.pool.GetHealthyClients()
	} else {
		// Get clients that handle any of the acceptable intents
		clientMap := make(map[string]*Client)
		for _, intent := range acceptableIntents {
			for _, c := range m.pool.GetClientsByIntent(intent) {
				clientMap[c.config.Name] = c
			}
		}
		for _, c := range clientMap {
			clients = append(clients, c)
		}
	}

	if len(clients) == 0 {
		log.Printf("[Federation] No healthy endpoints available for request %s", req.GetId())
		return &FederatedResponse{
			ID:             req.GetId(),
			Mutations:      nil,
			TotalLatencyMs: time.Since(startTime).Milliseconds(),
		}, nil
	}

	// Sort clients by priority
	sort.Slice(clients, func(i, j int) bool {
		return clients[i].config.Priority < clients[j].config.Priority
	})

	// Group clients by priority for parallel execution
	priorityGroups := groupByPriority(clients)

	var allMutations []*pb.Mutation
	var allResults []FederatedResult

	// Execute priority groups sequentially, endpoints within group in parallel
	for _, group := range priorityGroups {
		groupMutations, groupResults := m.executeGroup(ctx, req, group)
		allMutations = append(allMutations, groupMutations...)
		allResults = append(allResults, groupResults...)
	}

	// Filter mutations by acceptable intents if specified
	if len(acceptableIntents) > 0 {
		allMutations = filterMutationsByIntent(allMutations, acceptableIntents)
	}

	response := &FederatedResponse{
		ID:              req.GetId(),
		Mutations:       allMutations,
		EndpointResults: allResults,
		TotalLatencyMs:  time.Since(startTime).Milliseconds(),
		Metadata: &pb.Metadata{
			ApiVersion:   stringPtr("1.0"),
			ModelVersion: stringPtr("federated"),
		},
	}

	log.Printf("[Federation] Request %s completed in %dms, %d mutations from %d endpoints",
		req.GetId(), response.TotalLatencyMs, len(allMutations), len(allResults))

	return response, nil
}

// executeGroup executes all clients in a priority group in parallel
func (m *Manager) executeGroup(ctx context.Context, req *pb.RTBRequest, clients []*Client) ([]*pb.Mutation, []FederatedResult) {
	var wg sync.WaitGroup
	resultChan := make(chan FederatedResult, len(clients))

	for _, client := range clients {
		wg.Add(1)
		go func(c *Client) {
			defer wg.Done()
			result := m.executeClient(ctx, req, c)
			resultChan <- result
		}(client)
	}

	// Wait for all calls to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var mutations []*pb.Mutation
	var results []FederatedResult

	for result := range resultChan {
		results = append(results, result)
		if result.Success {
			mutations = append(mutations, result.Mutations...)
		}
	}

	return mutations, results
}

// executeClient executes a single client call
func (m *Manager) executeClient(ctx context.Context, req *pb.RTBRequest, client *Client) FederatedResult {
	startTime := time.Now()
	result := FederatedResult{
		EndpointName: client.config.Name,
	}

	resp, err := client.GetMutations(ctx, req)
	result.LatencyMs = time.Since(startTime).Milliseconds()

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		log.Printf("[Federation] Endpoint '%s' failed in %dms: %v",
			client.config.Name, result.LatencyMs, err)
	} else {
		result.Success = true
		result.Mutations = resp.GetMutations()
		log.Printf("[Federation] Endpoint '%s' returned %d mutations in %dms",
			client.config.Name, len(result.Mutations), result.LatencyMs)
	}

	return result
}

// CallEndpoint calls a specific endpoint by name
func (m *Manager) CallEndpoint(ctx context.Context, endpointName string, req *pb.RTBRequest) (*pb.RTBResponse, error) {
	client := m.pool.GetClient(endpointName)
	if client == nil {
		return nil, fmt.Errorf("endpoint '%s' not found", endpointName)
	}

	return client.GetMutations(ctx, req)
}

// ListEndpoints returns information about all configured endpoints
func (m *Manager) ListEndpoints() []EndpointInfo {
	var endpoints []EndpointInfo

	for _, ep := range m.config.Endpoints {
		client := m.pool.GetClient(ep.Name)
		healthy := client != nil && client.IsHealthy()

		info := EndpointInfo{
			Name:              ep.Name,
			Address:           ep.Address,
			Description:       ep.Description,
			Service:           ep.GetService(),
			ApplicableIntents: ep.ApplicableIntents,
			Priority:          ep.Priority,
			Enabled:           ep.IsEnabled(),
			Healthy:           healthy,
			TimeoutMs:         ep.GetTimeoutMs(m.config.Defaults),
		}
		endpoints = append(endpoints, info)
	}

	return endpoints
}

// GetEndpointInfo returns information about a specific endpoint
func (m *Manager) GetEndpointInfo(name string) (*EndpointInfo, error) {
	ep := m.config.GetEndpointByName(name)
	if ep == nil {
		return nil, fmt.Errorf("endpoint '%s' not found", name)
	}

	client := m.pool.GetClient(name)
	healthy := client != nil && client.IsHealthy()

	return &EndpointInfo{
		Name:              ep.Name,
		Address:           ep.Address,
		Description:       ep.Description,
		Service:           ep.GetService(),
		ApplicableIntents: ep.ApplicableIntents,
		Priority:          ep.Priority,
		Enabled:           ep.IsEnabled(),
		Healthy:           healthy,
		TimeoutMs:         ep.GetTimeoutMs(m.config.Defaults),
	}, nil
}

// Close shuts down the federation manager
func (m *Manager) Close() error {
	return m.pool.Close()
}

// Pool returns the underlying client pool
func (m *Manager) Pool() *ClientPool {
	return m.pool
}

// Config returns the federation configuration
func (m *Manager) Config() *Config {
	return m.config
}

// groupByPriority groups clients by their priority level
func groupByPriority(clients []*Client) [][]*Client {
	if len(clients) == 0 {
		return nil
	}

	groups := make(map[int][]*Client)
	var priorities []int

	for _, c := range clients {
		priority := c.config.Priority
		if _, exists := groups[priority]; !exists {
			priorities = append(priorities, priority)
		}
		groups[priority] = append(groups[priority], c)
	}

	sort.Ints(priorities)

	var result [][]*Client
	for _, p := range priorities {
		result = append(result, groups[p])
	}

	return result
}

// filterMutationsByIntent filters mutations to only include acceptable intents
func filterMutationsByIntent(mutations []*pb.Mutation, acceptableIntents []string) []*pb.Mutation {
	if len(acceptableIntents) == 0 {
		return mutations
	}

	intentSet := make(map[string]bool)
	for _, intent := range acceptableIntents {
		intentSet[intent] = true
	}

	var filtered []*pb.Mutation
	for _, m := range mutations {
		intentName := m.GetIntent().String()
		if intentSet[intentName] {
			filtered = append(filtered, m)
		}
	}

	return filtered
}

func stringPtr(s string) *string {
	return &s
}
