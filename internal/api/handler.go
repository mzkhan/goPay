package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/mzkhan/gopay/internal/models"
	"github.com/mzkhan/gopay/internal/processor"
)

// Handler holds HTTP route handlers for the payment API.
type Handler struct {
	proc           *processor.Processor
	idempotencyMu  sync.Mutex
	idempotencyMap map[string]*models.TransactionResponse // in-memory for MVP
}

func NewHandler(proc *processor.Processor) *Handler {
	return &Handler{
		proc:           proc,
		idempotencyMap: make(map[string]*models.TransactionResponse),
	}
}

// RegisterRoutes wires up all endpoints.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/transactions", h.ProcessTransaction)
	mux.HandleFunc("GET /health", h.Health)
}

// Health is a liveness check.
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

// ProcessTransaction handles POST /v1/transactions.
func (h *Handler) ProcessTransaction(w http.ResponseWriter, r *http.Request) {
	var req models.TransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, models.ErrorResponse{
			Error: "invalid request body",
			Code:  "INVALID_JSON",
		})
		return
	}

	// Basic validation.
	if err := validateRequest(&req); err != "" {
		writeJSON(w, http.StatusUnprocessableEntity, models.ErrorResponse{
			Error: err,
			Code:  "VALIDATION_ERROR",
		})
		return
	}

	// Idempotency check.
	h.idempotencyMu.Lock()
	if cached, ok := h.idempotencyMap[req.IdempotencyKey]; ok {
		h.idempotencyMu.Unlock()
		writeJSON(w, http.StatusOK, cached)
		return
	}
	h.idempotencyMu.Unlock()

	// Process.
	resp, err := h.proc.Process(r.Context(), &req)
	if err != nil {
		slog.Error("processing error", "error", err)
		writeJSON(w, http.StatusInternalServerError, models.ErrorResponse{
			Error: "internal processing error",
			Code:  "INTERNAL_ERROR",
		})
		return
	}

	// Cache for idempotency.
	h.idempotencyMu.Lock()
	h.idempotencyMap[req.IdempotencyKey] = resp
	h.idempotencyMu.Unlock()

	status := http.StatusCreated
	if resp.Status == "error" {
		status = http.StatusBadGateway
	}
	writeJSON(w, status, resp)
}

func validateRequest(req *models.TransactionRequest) string {
	if req.IdempotencyKey == "" {
		return "idempotency_key is required"
	}
	switch req.Type {
	case "authorization", "sale", "capture", "void", "refund":
	default:
		return "type must be one of: authorization, sale, capture, void, refund"
	}
	if req.Amount <= 0 {
		return "amount must be greater than 0"
	}
	if len(req.Currency) != 3 {
		return "currency must be a 3-letter ISO code"
	}
	if req.MerchantID == "" {
		return "merchant_id is required"
	}
	if req.Instrument.Token == "" {
		return "instrument.token is required"
	}
	if req.Instrument.TokenType != "vault" && req.Instrument.TokenType != "network" {
		return "instrument.token_type must be 'vault' or 'network'"
	}
	if req.Instrument.ExpiryMonth < 1 || req.Instrument.ExpiryMonth > 12 {
		return "instrument.expiry_month must be 1-12"
	}
	if req.Instrument.ExpiryYear < 2025 {
		return "instrument.expiry_year is invalid"
	}
	if req.Instrument.TokenType == "network" && req.Instrument.Cryptogram == "" {
		return "instrument.cryptogram is required for network tokens"
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
