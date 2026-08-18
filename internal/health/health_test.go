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

package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestProbeReady(t *testing.T) {
	checker := NewChecker()
	checker.SetReady(true)
	mux := http.NewServeMux()
	mux.HandleFunc("/health/ready", checker.ReadinessHandler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := Probe(ctx, srv.URL); err != nil {
		t.Fatalf("Probe ready: %v", err)
	}
}

func TestProbeNotReady(t *testing.T) {
	checker := NewChecker()
	mux := http.NewServeMux()
	mux.HandleFunc("/health/ready", checker.ReadinessHandler)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := Probe(ctx, srv.URL); err == nil {
		t.Fatal("Probe expected error when not ready")
	}
}

func TestProbeUnreachable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	if err := Probe(ctx, "http://127.0.0.1:1"); err == nil {
		t.Fatal("Probe expected error for unreachable origin")
	}
}