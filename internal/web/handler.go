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

// Package web implements the web UI for ARTF testing
package web

import (
	"embed"
	"encoding/json"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

//go:embed static/*
var staticFiles embed.FS

//go:embed templates/*
var templateFiles embed.FS

// Handler provides HTTP handlers for the web interface
type Handler struct {
	mcpEndpoint string
	samples     map[string]Sample
	templates   *template.Template
}

// Sample represents a sample ORTB payload
type Sample struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Payload     map[string]interface{} `json:"payload"`
}

// NewHandler creates a new web handler
func NewHandler(mcpEndpoint string) (*Handler, error) {
	// Parse templates
	tmpl, err := template.ParseFS(templateFiles, "templates/*.html")
	if err != nil {
		return nil, err
	}

	h := &Handler{
		mcpEndpoint: mcpEndpoint,
		samples:     make(map[string]Sample),
		templates:   tmpl,
	}

	// Load default samples
	h.loadDefaultSamples()

	return h, nil
}

// loadDefaultSamples loads the built-in sample payloads
func (h *Handler) loadDefaultSamples() {
	h.samples["banner-basic"] = Sample{
		Name:        "Basic Banner Request",
		Description: "A simple banner ad request with user demographics",
		Payload: map[string]interface{}{
			"id":                 "sample-banner-001",
			"tmax":               100,
			"lifecycle":          "LIFECYCLE_PUBLISHER_BID_REQUEST",
			"originator":         map[string]interface{}{"type": "TYPE_SSP", "id": "ssp-example-001"},
			"applicable_intents": []string{"ACTIVATE_SEGMENTS", "ACTIVATE_DEALS"},
			"bid_request": map[string]interface{}{
				"id": "auction-123",
				"imp": []interface{}{
					map[string]interface{}{
						"id": "imp-1",
						"banner": map[string]interface{}{
							"w":   300,
							"h":   250,
							"pos": 1,
						},
						"bidfloor":    1.50,
						"bidfloorcur": "USD",
					},
				},
				"site": map[string]interface{}{
					"id":     "site-456",
					"domain": "example.com",
					"cat":    []string{"IAB1"},
					"page":   "https://example.com/article",
				},
				"user": map[string]interface{}{
					"id":     "user-789",
					"yob":    1990,
					"gender": "M",
					"data": []interface{}{
						map[string]interface{}{
							"id":   "data-provider-1",
							"name": "Example DMP",
							"segment": []interface{}{
								map[string]interface{}{
									"id":   "seg-sports",
									"name": "Sports Enthusiast",
								},
							},
						},
					},
				},
				"device": map[string]interface{}{
					"ua": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
					"ip": "192.168.1.1",
					"geo": map[string]interface{}{
						"country": "USA",
						"region":  "CA",
					},
				},
			},
		},
	}

	h.samples["video-deals"] = Sample{
		Name:        "Video Request with Deals",
		Description: "A video ad request with private marketplace deals",
		Payload: map[string]interface{}{
			"id":                 "sample-video-001",
			"tmax":               150,
			"lifecycle":          "LIFECYCLE_PUBLISHER_BID_REQUEST",
			"originator":         map[string]interface{}{"type": "TYPE_EXCHANGE", "id": "exchange-video-001"},
			"applicable_intents": []string{"ACTIVATE_SEGMENTS", "ACTIVATE_DEALS", "ADJUST_DEAL_FLOOR"},
			"bid_request": map[string]interface{}{
				"id": "auction-456",
				"imp": []interface{}{
					map[string]interface{}{
						"id": "imp-1",
						"video": map[string]interface{}{
							"mimes":       []string{"video/mp4"},
							"minduration": 15,
							"maxduration": 30,
							"w":           640,
							"h":           480,
						},
						"bidfloor": 8.00,
						"pmp": map[string]interface{}{
							"private_auction": true,
							"deals": []interface{}{
								map[string]interface{}{
									"id":       "deal-premium-video",
									"bidfloor": 10.00,
									"at":       1,
								},
							},
						},
					},
				},
				"site": map[string]interface{}{
					"id":     "site-789",
					"domain": "streaming.example.com",
					"cat":    []string{"IAB1-6"},
				},
				"user": map[string]interface{}{
					"id":  "user-456",
					"yob": 1985,
				},
			},
		},
	}

	h.samples["bid-shading"] = Sample{
		Name:        "Bid Response with Shading",
		Description: "A complete request/response pair for bid shading demonstration",
		Payload: map[string]interface{}{
			"id":                 "sample-bidshade-001",
			"tmax":               100,
			"lifecycle":          "LIFECYCLE_DSP_BID_RESPONSE",
			"originator":         map[string]interface{}{"type": "TYPE_DSP", "id": "dsp-bidshade-001"},
			"applicable_intents": []string{"BID_SHADE"},
			"bid_request": map[string]interface{}{
				"id": "auction-789",
				"imp": []interface{}{
					map[string]interface{}{
						"id": "imp-1",
						"banner": map[string]interface{}{
							"w": 728,
							"h": 90,
						},
						"bidfloor": 2.00,
					},
				},
				"user": map[string]interface{}{
					"id":  "user-123",
					"yob": 1975,
				},
			},
			"bid_response": map[string]interface{}{
				"id": "auction-789",
				"seatbid": []interface{}{
					map[string]interface{}{
						"seat": "dsp-001",
						"bid": []interface{}{
							map[string]interface{}{
								"id":      "bid-abc",
								"impid":   "imp-1",
								"price":   5.50,
								"adomain": []string{"advertiser.com"},
							},
						},
					},
				},
			},
		},
	}

	h.samples["native-ad"] = Sample{
		Name:        "Native Ad Request",
		Description: "A native advertising request",
		Payload: map[string]interface{}{
			"id":                 "sample-native-001",
			"tmax":               100,
			"lifecycle":          "LIFECYCLE_PUBLISHER_BID_REQUEST",
			"originator":         map[string]interface{}{"type": "TYPE_SSP", "id": "ssp-mobile-001"},
			"applicable_intents": []string{"ACTIVATE_SEGMENTS", "ACTIVATE_DEALS"},
			"bid_request": map[string]interface{}{
				"id": "auction-native-123",
				"imp": []interface{}{
					map[string]interface{}{
						"id": "imp-1",
						"native": map[string]interface{}{
							"request": `{"ver":"1.2","assets":[{"id":1,"required":1,"title":{"len":90}}]}`,
							"ver":     "1.2",
						},
						"bidfloor": 3.00,
					},
				},
				"app": map[string]interface{}{
					"id":     "app-123",
					"name":   "Example App",
					"bundle": "com.example.app",
					"cat":    []string{"IAB9"},
				},
				"user": map[string]interface{}{
					"id":  "user-mobile-456",
					"yob": 2000,
				},
				"device": map[string]interface{}{
					"ua":         "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)",
					"devicetype": 4,
					"os":         "iOS",
					"osv":        "17.0",
				},
			},
		},
	}

	h.samples["rust-federation"] = Sample{
		Name:        "Rust Agent Federation",
		Description: "Test federation with the Rust RTB agent (auction-456 with ACTIVATE_SEGMENTS and ACTIVATE_DEALS)",
		Payload: map[string]interface{}{
			"id":                  "auction-456",
			"tmax":                500,
			"lifecycle":           "LIFECYCLE_PUBLISHER_BID_REQUEST",
			"originator":          map[string]interface{}{"type": "TYPE_SSP", "id": "ssp-federation-001"},
			"applicable_intents":  []string{"ACTIVATE_SEGMENTS", "ACTIVATE_DEALS"},
			"federate":            true,
			"federate_endpoints":  []string{"rust-rtb-agent"},
			"bid_request": map[string]interface{}{
				"id": "auction-456",
				"imp": []interface{}{
					map[string]interface{}{
						"id": "imp-1",
						"video": map[string]interface{}{
							"mimes":       []string{"video/mp4"},
							"minduration": 15,
							"maxduration": 30,
							"w":           640,
							"h":           480,
						},
						"bidfloor":    8.00,
						"bidfloorcur": "USD",
					},
				},
				"site": map[string]interface{}{
					"id":     "site-789",
					"domain": "streaming.example.com",
					"cat":    []string{"IAB1-6"},
					"page":   "https://streaming.example.com/live",
				},
				"user": map[string]interface{}{
					"id":     "user-456",
					"yob":    1985,
					"gender": "F",
					"data": []interface{}{
						map[string]interface{}{
							"id":   "data-provider-2",
							"name": "Example DMP",
							"segment": []interface{}{
								map[string]interface{}{
									"id":   "seg-video",
									"name": "Video Enthusiast",
								},
							},
						},
					},
				},
				"device": map[string]interface{}{
					"ua": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36",
					"ip": "10.0.0.1",
					"geo": map[string]interface{}{
						"country": "USA",
						"region":  "NY",
					},
				},
			},
		},
	}

	h.samples["multi-imp"] = Sample{
		Name:        "Multi-Impression Request",
		Description: "A request with multiple impression opportunities",
		Payload: map[string]interface{}{
			"id":                 "sample-multi-001",
			"tmax":               120,
			"lifecycle":          "LIFECYCLE_PUBLISHER_BID_REQUEST",
			"originator":         map[string]interface{}{"type": "TYPE_PUBLISHER", "id": "pub-news-001"},
			"applicable_intents": []string{"ACTIVATE_SEGMENTS", "ACTIVATE_DEALS", "ADJUST_DEAL_FLOOR"},
			"bid_request": map[string]interface{}{
				"id": "auction-multi-123",
				"imp": []interface{}{
					map[string]interface{}{
						"id": "imp-header",
						"banner": map[string]interface{}{
							"w": 970,
							"h": 250,
						},
						"bidfloor": 4.00,
					},
					map[string]interface{}{
						"id": "imp-sidebar",
						"banner": map[string]interface{}{
							"w": 300,
							"h": 600,
						},
						"bidfloor": 2.50,
					},
					map[string]interface{}{
						"id": "imp-footer",
						"banner": map[string]interface{}{
							"w": 728,
							"h": 90,
						},
						"bidfloor": 1.00,
					},
				},
				"site": map[string]interface{}{
					"id":     "site-news",
					"domain": "news.example.com",
					"cat":    []string{"IAB12"},
				},
				"user": map[string]interface{}{
					"id":     "user-news-reader",
					"yob":    1988,
					"gender": "F",
				},
			},
		},
	}
}

// RegisterRoutes registers the web routes with the given mux
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// Serve static files
	staticFS, _ := fs.Sub(staticFiles, "static")
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// API routes
	mux.HandleFunc("/api/samples", h.handleListSamples)
	mux.HandleFunc("/api/samples/", h.handleGetSample)

	// Specification page
	mux.HandleFunc("/spec", h.handleSpec)

	// Container guide page
	mux.HandleFunc("/container", h.handleContainer)

	// Main page
	mux.HandleFunc("/", h.handleIndex)
}

// handleSpec serves the ARTF specification page
func (h *Handler) handleSpec(w http.ResponseWriter, r *http.Request) {
	specFile, err := staticFiles.ReadFile("static/spec.html")
	if err != nil {
		http.Error(w, "Specification not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(specFile)
}

// handleContainer serves the container guide page
func (h *Handler) handleContainer(w http.ResponseWriter, r *http.Request) {
	containerFile, err := staticFiles.ReadFile("static/container.html")
	if err != nil {
		http.Error(w, "Container guide not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(containerFile)
}

// handleIndex serves the main page
func (h *Handler) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := struct {
		MCPEndpoint string
		Samples     map[string]Sample
	}{
		MCPEndpoint: h.mcpEndpoint,
		Samples:     h.samples,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		log.Printf("Template error: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// handleListSamples returns the list of available samples
func (h *Handler) handleListSamples(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sampleList := make([]map[string]string, 0, len(h.samples))
	for id, sample := range h.samples {
		sampleList = append(sampleList, map[string]string{
			"id":          id,
			"name":        sample.Name,
			"description": sample.Description,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sampleList)
}

// handleGetSample returns a specific sample payload
func (h *Handler) handleGetSample(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract sample ID from path
	sampleID := strings.TrimPrefix(r.URL.Path, "/api/samples/")
	sampleID = filepath.Clean(sampleID)

	sample, ok := h.samples[sampleID]
	if !ok {
		http.Error(w, "Sample not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sample.Payload)
}

// AddSample adds a custom sample payload
func (h *Handler) AddSample(id string, sample Sample) {
	h.samples[id] = sample
}
