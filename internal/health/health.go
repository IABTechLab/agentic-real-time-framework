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

// Package health implements Kubernetes-compatible health check endpoints
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Checker implements liveness and readiness probes
type Checker struct {
	mu    sync.RWMutex
	ready bool
}

// HealthResponse is the JSON response for health endpoints
type HealthResponse struct {
	Status  string `json:"status"`
	Ready   bool   `json:"ready,omitempty"`
	Version string `json:"version,omitempty"`
}

// NewChecker creates a new health checker
func NewChecker() *Checker {
	return &Checker{
		ready: false,
	}
}

// SetReady sets the readiness state
func (c *Checker) SetReady(ready bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ready = ready
}

// IsReady returns the current readiness state
func (c *Checker) IsReady() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ready
}

// LivenessHandler handles /health/live requests
// Returns 200 if the process is alive
func (c *Checker) LivenessHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:  "alive",
		Version: "0.10.0",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ReadinessHandler handles /health/ready requests
// Returns 200 if the server is ready to accept traffic, 503 otherwise
func (c *Checker) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	ready := c.IsReady()

	response := HealthResponse{
		Ready:   ready,
		Version: "0.10.0",
	}

	w.Header().Set("Content-Type", "application/json")

	if ready {
		response.Status = "ready"
		w.WriteHeader(http.StatusOK)
	} else {
		response.Status = "not_ready"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}

// Probe GETs origin/health/ready. Used by the process-local -health-check flag
// so Docker HEALTHCHECK does not need curl or wget in the image.
func Probe(ctx context.Context, origin string) error {
	u := strings.TrimRight(origin, "/") + "/health/ready"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s returned %s", u, resp.Status)
	}
	return nil
}
