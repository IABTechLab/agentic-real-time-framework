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

// Package federation provides federated GRPC endpoint management
package federation

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the federation configuration
type Config struct {
	// Version of the config schema
	Version string `json:"version" yaml:"version"`

	// Endpoints is a list of federated GRPC endpoints
	Endpoints []EndpointConfig `json:"endpoints" yaml:"endpoints"`

	// Defaults for all endpoints
	Defaults *EndpointDefaults `json:"defaults,omitempty" yaml:"defaults,omitempty"`
}

// EndpointDefaults contains default settings for endpoints
type EndpointDefaults struct {
	// TimeoutMs is the default timeout in milliseconds
	TimeoutMs int `json:"timeout_ms,omitempty" yaml:"timeout_ms,omitempty"`

	// MaxRetries is the default number of retries
	MaxRetries int `json:"max_retries,omitempty" yaml:"max_retries,omitempty"`

	// TLS configuration
	TLS *TLSConfig `json:"tls,omitempty" yaml:"tls,omitempty"`
}

// EndpointConfig represents a single federated GRPC endpoint
type EndpointConfig struct {
	// Name is a unique identifier for this endpoint
	Name string `json:"name" yaml:"name"`

	// Address is the GRPC address (host:port)
	Address string `json:"address" yaml:"address"`

	// Description provides human-readable info about this endpoint
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Service is the GRPC service type (only "RTBExtensionPoint" is supported)
	Service string `json:"service,omitempty" yaml:"service,omitempty"`

	// ApplicableIntents is the list of intents this endpoint can handle (IAB spec field name)
	// If empty, all intents are applicable
	ApplicableIntents []string `json:"applicable_intents" yaml:"applicable_intents"`

	// Priority determines call order (lower = higher priority, called first)
	// Endpoints with same priority may be called in parallel
	Priority int `json:"priority,omitempty" yaml:"priority,omitempty"`

	// Enabled determines if this endpoint is active
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`

	// TimeoutMs overrides the default timeout for this endpoint
	TimeoutMs int `json:"timeout_ms,omitempty" yaml:"timeout_ms,omitempty"`

	// MaxRetries overrides the default retries for this endpoint
	MaxRetries int `json:"max_retries,omitempty" yaml:"max_retries,omitempty"`

	// TLS configuration for this endpoint
	TLS *TLSConfig `json:"tls,omitempty" yaml:"tls,omitempty"`

	// Metadata is arbitrary key-value data for this endpoint
	Metadata map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// HealthCheck configuration
	HealthCheck *HealthCheckConfig `json:"health_check,omitempty" yaml:"health_check,omitempty"`
}

// TLSConfig contains TLS settings
type TLSConfig struct {
	// Enabled determines if TLS is used
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Insecure skips certificate verification (for testing)
	Insecure bool `json:"insecure,omitempty" yaml:"insecure,omitempty"`

	// CertFile is the path to the client certificate
	CertFile string `json:"cert_file,omitempty" yaml:"cert_file,omitempty"`

	// KeyFile is the path to the client key
	KeyFile string `json:"key_file,omitempty" yaml:"key_file,omitempty"`

	// CAFile is the path to the CA certificate
	CAFile string `json:"ca_file,omitempty" yaml:"ca_file,omitempty"`

	// ServerName overrides the server name for verification
	ServerName string `json:"server_name,omitempty" yaml:"server_name,omitempty"`
}

// HealthCheckConfig contains health check settings
type HealthCheckConfig struct {
	// Enabled determines if health checking is active
	Enabled bool `json:"enabled" yaml:"enabled"`

	// IntervalSeconds is the time between health checks
	IntervalSeconds int `json:"interval_seconds,omitempty" yaml:"interval_seconds,omitempty"`

	// TimeoutSeconds is the health check timeout
	TimeoutSeconds int `json:"timeout_seconds,omitempty" yaml:"timeout_seconds,omitempty"`
}

// IsEnabled returns whether this endpoint is enabled (default: true)
func (e *EndpointConfig) IsEnabled() bool {
	if e.Enabled == nil {
		return true
	}
	return *e.Enabled
}

// GetService returns the service type (default: RTBExtensionPoint)
func (e *EndpointConfig) GetService() string {
	if e.Service == "" {
		return "RTBExtensionPoint"
	}
	return e.Service
}

// GetTimeoutMs returns the timeout in milliseconds (default: 100)
func (e *EndpointConfig) GetTimeoutMs(defaults *EndpointDefaults) int {
	if e.TimeoutMs > 0 {
		return e.TimeoutMs
	}
	if defaults != nil && defaults.TimeoutMs > 0 {
		return defaults.TimeoutMs
	}
	return 100 // Default 100ms for RTB latency requirements
}

// GetMaxRetries returns the max retries (default: 0)
func (e *EndpointConfig) GetMaxRetries(defaults *EndpointDefaults) int {
	if e.MaxRetries > 0 {
		return e.MaxRetries
	}
	if defaults != nil && defaults.MaxRetries > 0 {
		return defaults.MaxRetries
	}
	return 0
}

// HasIntent checks if this endpoint accepts the given intent
func (e *EndpointConfig) HasIntent(intent string) bool {
	if len(e.ApplicableIntents) == 0 {
		return true // All intents applicable
	}
	for _, i := range e.ApplicableIntents {
		if strings.EqualFold(i, intent) {
			return true
		}
	}
	return false
}

// LoadConfig loads federation configuration from a file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return ParseConfig(data, path)
}

// ParseConfig parses configuration from bytes
func ParseConfig(data []byte, filename string) (*Config, error) {
	var config Config

	// Determine format by extension or try both
	if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML config: %w", err)
		}
	} else if strings.HasSuffix(filename, ".json") {
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON config: %w", err)
		}
	} else {
		// Try YAML first, then JSON
		if err := yaml.Unmarshal(data, &config); err != nil {
			if err := json.Unmarshal(data, &config); err != nil {
				return nil, fmt.Errorf("failed to parse config (tried YAML and JSON)")
			}
		}
	}

	// Validate config
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate checks the configuration for errors
func (c *Config) Validate() error {
	seen := make(map[string]bool)
	for i, ep := range c.Endpoints {
		if ep.Name == "" {
			return fmt.Errorf("endpoint %d: name is required", i)
		}
		if seen[ep.Name] {
			return fmt.Errorf("endpoint %d: duplicate name '%s'", i, ep.Name)
		}
		seen[ep.Name] = true

		if ep.Address == "" {
			return fmt.Errorf("endpoint '%s': address is required", ep.Name)
		}

		// Only RTBExtensionPoint is supported
		if ep.GetService() != "RTBExtensionPoint" {
			return fmt.Errorf("endpoint '%s': unsupported service type '%s' (only RTBExtensionPoint is supported)", ep.Name, ep.Service)
		}

		// Validate intents
		for _, intent := range ep.ApplicableIntents {
			if !isValidIntent(intent) {
				return fmt.Errorf("endpoint '%s': invalid intent '%s'", ep.Name, intent)
			}
		}
	}

	return nil
}

// GetEnabledEndpoints returns only enabled endpoints
func (c *Config) GetEnabledEndpoints() []EndpointConfig {
	var enabled []EndpointConfig
	for _, ep := range c.Endpoints {
		if ep.IsEnabled() {
			enabled = append(enabled, ep)
		}
	}
	return enabled
}

// GetEndpointsByIntent returns endpoints that accept the given intent
func (c *Config) GetEndpointsByIntent(intent string) []EndpointConfig {
	var matching []EndpointConfig
	for _, ep := range c.Endpoints {
		if ep.IsEnabled() && ep.HasIntent(intent) {
			matching = append(matching, ep)
		}
	}
	return matching
}

// GetEndpointByName returns an endpoint by name
func (c *Config) GetEndpointByName(name string) *EndpointConfig {
	for i := range c.Endpoints {
		if c.Endpoints[i].Name == name {
			return &c.Endpoints[i]
		}
	}
	return nil
}

// ValidIntents is the list of valid ARTF intent names
var ValidIntents = []string{
	"ACTIVATE_SEGMENTS",
	"ACTIVATE_DEALS",
	"SUPPRESS_DEALS",
	"ADJUST_DEAL_FLOOR",
	"ADJUST_DEAL_MARGIN",
	"BID_SHADE",
	"ADD_METRICS",
	"ADD_CIDS",
}

func isValidIntent(intent string) bool {
	for _, valid := range ValidIntents {
		if strings.EqualFold(intent, valid) {
			return true
		}
	}
	return false
}
