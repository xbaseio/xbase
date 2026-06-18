package observability

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/xbaseio/xbase/component"
	"github.com/xbaseio/xbase/locate"
	"github.com/xbaseio/xbase/registry"
)

type DebugServer struct {
	component.Base
	addr     string
	locator  locate.Locator
	registry registry.Registry
	server   *http.Server
}

type UserLocateResponse struct {
	UID   int64             `json:"uid"`
	Gate  string            `json:"gate"`
	Node  string            `json:"node,omitempty"`
	Nodes map[string]string `json:"nodes"`
}

func NewDebugServer(addr string, locator locate.Locator, reg registry.Registry) *DebugServer {
	return &DebugServer{
		addr:     addr,
		locator:  locator,
		registry: reg,
	}
}

func (s *DebugServer) Name() string {
	return "cluster-demo-debug"
}

func (s *DebugServer) Start() {
	if s.addr == "" {
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/debug/user", s.handleUser)
	mux.HandleFunc("/debug/services", s.handleServices)

	s.server = &http.Server{
		Addr:              s.addr,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
	}

	go func() {
		Info("debug_server_start", "addr", s.addr)
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			Error("debug_server_crash", err, "addr", s.addr)
		}
	}()
}

func (s *DebugServer) Close() {
	if s.server == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		Warn("debug_server_shutdown_failed", err, "addr", s.addr)
	}
}

func (s *DebugServer) handleHealthz(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]any{
		"ok":   true,
		"addr": s.addr,
	})
}

func (s *DebugServer) handleUser(w http.ResponseWriter, r *http.Request) {
	if s.locator == nil {
		s.writeJSON(w, http.StatusNotImplemented, map[string]any{
			"error": "locator is not configured",
		})
		return
	}

	uid, err := strconv.ParseInt(r.URL.Query().Get("uid"), 10, 64)
	if err != nil || uid <= 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "invalid uid",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	gid, err := s.locator.LocateGate(ctx, uid)
	if err != nil {
		Error("debug_user_locate_gate_failed", err, "uid", uid)
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	nodes, err := s.locator.LocateNodes(ctx, uid)
	if err != nil {
		Error("debug_user_locate_nodes_failed", err, "uid", uid)
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	resp := UserLocateResponse{
		UID:   uid,
		Gate:  gid,
		Nodes: nodes,
	}

	if nodeName := r.URL.Query().Get("node"); nodeName != "" {
		resp.Node = nodes[nodeName]
	}

	Info("debug_user_lookup", "uid", uid, "gate", gid, "nodes", len(nodes))
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *DebugServer) handleServices(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		s.writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "missing service name",
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	services, err := s.registry.Services(ctx, name)
	if err != nil {
		Error("debug_services_failed", err, "name", name)
		s.writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	Info("debug_services_lookup", "name", name, "count", len(services))
	s.writeJSON(w, http.StatusOK, services)
}

func (s *DebugServer) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
