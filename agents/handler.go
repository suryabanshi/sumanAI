package agents

import (
	"encoding/json"
	"io"
	"net/http"
)

// Handler exposes the routine-generation service over HTTP.
type Handler struct {
	svc *Service
}

// NewHandler builds an HTTP handler for the routine service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes wires the agent/routine endpoints onto a mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/agents", h.ListAgents)
	mux.HandleFunc("POST /api/v1/agents/{slug}/routines", h.GenerateRoutine)
	mux.HandleFunc("POST /api/v1/routines/generate-all", h.GenerateAll)
}

// ListAgents returns the catalog of profession agents.
func (h *Handler) ListAgents(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"professions": h.svc.Registry().Professions()})
}

// GenerateRoutine generates a routine for one profession.
func (h *Handler) GenerateRoutine(w http.ResponseWriter, r *http.Request) {
	opts, err := decodeOptions(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	routine, err := h.svc.GenerateRoutine(r.PathValue("slug"), opts)
	if err == ErrProfessionNotFound {
		writeError(w, http.StatusNotFound, "profession not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, routine)
}

// GenerateAll generates routines for all professionals in one call.
func (h *Handler) GenerateAll(w http.ResponseWriter, r *http.Request) {
	opts, err := decodeOptions(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	routines, err := h.svc.GenerateForAll(opts)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"routines": routines, "count": len(routines)})
}

// decodeOptions reads GenerateOptions from the request body, tolerating an empty body.
func decodeOptions(r *http.Request) (GenerateOptions, error) {
	var opts GenerateOptions
	if r.Body == nil || r.ContentLength == 0 {
		return opts, nil
	}
	err := json.NewDecoder(r.Body).Decode(&opts)
	if err == io.EOF {
		return GenerateOptions{}, nil
	}
	return opts, err
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
