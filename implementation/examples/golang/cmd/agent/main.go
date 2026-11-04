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

// Package main implements the ARTF agent with gRPC, MCP, and Web interfaces
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/iabtechlab/agentic-rtb-framework/examples/golang/internal/agent"
	"github.com/iabtechlab/agentic-rtb-framework/examples/golang/internal/federation"
	"github.com/iabtechlab/agentic-rtb-framework/examples/golang/internal/handlers"
	"github.com/iabtechlab/agentic-rtb-framework/examples/golang/internal/health"
	"github.com/iabtechlab/agentic-rtb-framework/examples/golang/internal/mcp"
	"github.com/iabtechlab/agentic-rtb-framework/examples/golang/internal/web"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Version information (set at build time)
var (
	Version   = "dev"
	BuildTime = "unknown"
)

// Command-line flags
var (
	// Feature flags
	enableGRPC = flag.Bool("enable-grpc", true, "Enable gRPC interface")
	enableMCP  = flag.Bool("enable-mcp", false, "Enable MCP interface")
	enableWeb  = flag.Bool("enable-web", false, "Enable web interface")

	// Bind address for all services (use 0.0.0.0 to bind to all interfaces, 127.0.0.1 for localhost only)
	listenAddr = flag.String("listen", "", "Bind address for all services (default: all interfaces)")

	// External URL for load balancer scenarios (e.g., https://api.example.com)
	externalURL = flag.String("external-url", "", "External base URL for load balancer (rewrites all service URLs)")

	// Port configuration (optional, with defaults)
	grpcPort   = flag.Int("grpc-port", 50051, "gRPC port")
	mcpPort    = flag.Int("mcp-port", 50052, "MCP port")
	webPort    = flag.Int("web-port", 8081, "Web interface port")
	healthPort = flag.Int("health-port", 8080, "Health check HTTP port")

	// Federation configuration
	federationConfig = flag.String("federation-config", "", "Path to federation configuration file (YAML/JSON)")

	// Version flag
	showVersion = flag.Bool("version", false, "Show version information")
)

func main() {
	flag.Parse()

	// Show version and exit
	if *showVersion {
		fmt.Printf("ARTF Agent v%s (built: %s)\n", Version, BuildTime)
		os.Exit(0)
	}

	log.Printf("Starting ARTF Agent v%s", Version)
	log.Printf("Features: gRPC=%v, MCP=%v, Web=%v", *enableGRPC, *enableMCP, *enableWeb)

	// Validate that at least one service is enabled
	if !*enableGRPC && !*enableMCP && !*enableWeb {
		log.Fatal("At least one service must be enabled (--enable-grpc, --enable-mcp, or --enable-web)")
	}

	// Create shared components
	healthChecker := health.NewChecker()
	mutationHandlers := handlers.NewMutationHandlers()

	// Create the ARTF agent (shared by both gRPC and MCP interfaces)
	// This ensures a single implementation for all business logic
	artfAgent := agent.NewARTFAgent(mutationHandlers)

	// Track services for shutdown
	var grpcServer *grpc.Server
	var mcpAgent *mcp.Agent
	var webServer *http.Server
	var healthServer *http.Server

	// Start gRPC interface
	if *enableGRPC {
		grpcServer = grpc.NewServer(
			grpc.UnaryInterceptor(agent.LoggingInterceptor),
		)

		agent.RegisterRTBExtensionPointServer(grpcServer, artfAgent)
		reflection.Register(grpcServer)

		grpcListenAddr := fmt.Sprintf("%s:%d", *listenAddr, *grpcPort)
		grpcListener, err := net.Listen("tcp", grpcListenAddr)
		if err != nil {
			log.Fatalf("Failed to listen on gRPC address %s: %v", grpcListenAddr, err)
		}

		go func() {
			log.Printf("gRPC interface listening on %s", grpcListenAddr)
			if err := grpcServer.Serve(grpcListener); err != nil {
				log.Printf("gRPC error: %v", err)
			}
		}()
	}

	// Initialize federation manager if configured
	var federationManager *federation.Manager
	if *federationConfig != "" {
		fm, err := federation.NewManagerFromFile(*federationConfig)
		if err != nil {
			log.Printf("Warning: Failed to initialize federation manager: %v", err)
		} else {
			federationManager = fm
			log.Printf("Federation manager initialized from %s", *federationConfig)
		}
	}

	// When both Web and MCP are enabled, serve them on the same port (web port)
	// This allows an external load balancer to route to a single endpoint
	if *enableWeb && *enableMCP {
		// Create MCP agent that wraps the gRPC agent (single implementation)
		mcpAgent = mcp.NewAgent(artfAgent, *listenAddr, *webPort)

		// Attach federation manager if configured
		if federationManager != nil {
			mcpAgent.SetFederationManager(federationManager)
			log.Printf("Federation manager attached to MCP interface")
		}

		// MCP endpoint is relative when served on same port
		mcpEndpoint := buildMCPEndpoint()

		webHandler, err := web.NewHandler(mcpEndpoint)
		if err != nil {
			log.Fatalf("Failed to create web handler: %v", err)
		}

		// Create unified mux with both Web and MCP routes
		webMux := http.NewServeMux()
		webHandler.RegisterRoutes(webMux)
		// Mount MCP handler at /mcp path
		webMux.Handle("/mcp", mcpAgent.Handler())

		webListenAddr := fmt.Sprintf("%s:%d", *listenAddr, *webPort)
		webServer = &http.Server{
			Addr:         webListenAddr,
			Handler:      webMux,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		}

		go func() {
			log.Printf("Web + MCP interface listening on %s", webListenAddr)
			if err := webServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("Web + MCP interface error: %v", err)
			}
		}()
	} else {
		// Start MCP interface standalone
		if *enableMCP {
			// MCP agent wraps the gRPC agent (single implementation)
			mcpAgent = mcp.NewAgent(artfAgent, *listenAddr, *mcpPort)

			// Attach federation manager if configured
			if federationManager != nil {
				mcpAgent.SetFederationManager(federationManager)
				log.Printf("Federation manager attached to MCP interface")
			}

			mcpListenAddr := fmt.Sprintf("%s:%d", *listenAddr, *mcpPort)

			go func() {
				log.Printf("MCP interface listening on %s", mcpListenAddr)
				if err := mcpAgent.Start(); err != nil {
					log.Printf("MCP error: %v", err)
				}
			}()
		}

		// Start Web interface standalone
		if *enableWeb {
			mcpEndpoint := "/mcp-disabled"

			webHandler, err := web.NewHandler(mcpEndpoint)
			if err != nil {
				log.Fatalf("Failed to create web handler: %v", err)
			}

			webMux := http.NewServeMux()
			webHandler.RegisterRoutes(webMux)

			webListenAddr := fmt.Sprintf("%s:%d", *listenAddr, *webPort)
			webServer = &http.Server{
				Addr:         webListenAddr,
				Handler:      webMux,
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
			}

			go func() {
				log.Printf("Web interface listening on %s", webListenAddr)
				if err := webServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Printf("Web interface error: %v", err)
				}
			}()
		}
	}

	// Start health check HTTP endpoint (always enabled)
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health/live", healthChecker.LivenessHandler)
	healthMux.HandleFunc("/health/ready", healthChecker.ReadinessHandler)
	healthMux.HandleFunc("/health/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		externalURLVal := ""
		if *externalURL != "" {
			externalURLVal = *externalURL
		}
		fmt.Fprintf(w, `{"version":"%s","build_time":"%s","grpc":%v,"mcp":%v,"web":%v,"external_url":"%s"}`,
			Version, BuildTime, *enableGRPC, *enableMCP, *enableWeb, externalURLVal)
	})

	healthListenAddr := fmt.Sprintf("%s:%d", *listenAddr, *healthPort)
	healthServer = &http.Server{
		Addr:         healthListenAddr,
		Handler:      healthMux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("Health check endpoint listening on %s", healthListenAddr)
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Health check error: %v", err)
		}
	}()

	// Mark agent as ready
	healthChecker.SetReady(true)
	log.Printf("Agent is ready to accept requests")

	// Print service URLs
	printServiceURLs()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Printf("Shutting down agent...")

	// Mark as not ready during shutdown
	healthChecker.SetReady(false)

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown services
	if grpcServer != nil {
		grpcServer.GracefulStop()
		log.Printf("gRPC interface stopped")
	}

	if webServer != nil {
		if err := webServer.Shutdown(ctx); err != nil {
			log.Printf("Web interface shutdown error: %v", err)
		} else {
			log.Printf("Web interface stopped")
		}
	}

	if federationManager != nil {
		if err := federationManager.Close(); err != nil {
			log.Printf("Federation manager shutdown error: %v", err)
		} else {
			log.Printf("Federation manager stopped")
		}
	}

	if err := healthServer.Shutdown(ctx); err != nil {
		log.Printf("Health endpoint shutdown error: %v", err)
	}

	log.Printf("Agent stopped")
}

// printServiceURLs prints the URLs for enabled services
func printServiceURLs() {
	log.Println("=== Service URLs ===")

	if *externalURL != "" {
		log.Printf("  External URL: %s", *externalURL)
	}

	if *enableGRPC {
		if *externalURL != "" {
			log.Printf("  gRPC:   %s (external)", buildExternalGRPCAddr())
		} else {
			log.Printf("  gRPC:   %s:%d", formatDisplayAddr(*listenAddr), *grpcPort)
		}
	}

	// When both Web and MCP are enabled, they share the same port
	if *enableWeb && *enableMCP {
		log.Printf("  Web+MCP: %s", buildWebEndpoint())
		log.Printf("    └─ MCP: %s", buildMCPEndpoint())
	} else {
		if *enableMCP {
			log.Printf("  MCP:    %s", buildMCPEndpoint())
		}
		if *enableWeb {
			log.Printf("  Web UI: %s", buildWebEndpoint())
		}
	}

	log.Printf("  Health: %s", buildHealthEndpoint())
	log.Println("====================")
}

// formatDisplayAddr returns a display-friendly address (localhost if empty)
func formatDisplayAddr(addr string) string {
	if addr == "" || addr == "0.0.0.0" {
		return "localhost"
	}
	return addr
}

// buildMCPEndpoint returns the MCP endpoint URL, using external URL if configured.
// When both web and MCP are enabled, MCP is served on the web port.
func buildMCPEndpoint() string {
	if *externalURL != "" {
		return buildExternalURL("/mcp")
	}
	// When both web and MCP are enabled, they share the web port
	port := *mcpPort
	if *enableWeb && *enableMCP {
		port = *webPort
	}
	return fmt.Sprintf("http://%s:%d/mcp", formatDisplayAddr(*listenAddr), port)
}

// buildWebEndpoint returns the Web UI endpoint URL, using external URL if configured
func buildWebEndpoint() string {
	if *externalURL != "" {
		return buildExternalURL("/")
	}
	return fmt.Sprintf("http://%s:%d/", formatDisplayAddr(*listenAddr), *webPort)
}

// buildHealthEndpoint returns the health check endpoint URL, using external URL if configured
func buildHealthEndpoint() string {
	if *externalURL != "" {
		return buildExternalURL("/health/ready")
	}
	return fmt.Sprintf("http://%s:%d/health/ready", formatDisplayAddr(*listenAddr), *healthPort)
}

// buildExternalGRPCAddr returns the external gRPC address
func buildExternalGRPCAddr() string {
	if *externalURL != "" {
		parsed, err := url.Parse(*externalURL)
		if err == nil {
			// For gRPC, we typically use just the host (without path)
			// The port might be different or handled by the load balancer
			host := parsed.Host
			if !strings.Contains(host, ":") {
				// Add default gRPC port if not specified
				return fmt.Sprintf("%s:%d", host, *grpcPort)
			}
			return host
		}
	}
	return fmt.Sprintf("%s:%d", formatDisplayAddr(*listenAddr), *grpcPort)
}

// buildExternalURL constructs a URL using the external base URL and the given path
func buildExternalURL(path string) string {
	baseURL := strings.TrimSuffix(*externalURL, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return baseURL + path
}
