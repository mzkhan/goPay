package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mzkhan/gopay/internal/api"
	"github.com/mzkhan/gopay/internal/models"
	"github.com/mzkhan/gopay/internal/processor"
	"github.com/mzkhan/gopay/internal/token"
)

func newTestHandler() *api.Handler {
	vault := &token.StubVault{}
	sender := processor.NewSimulatorSender()
	proc := processor.New(vault, sender)
	return api.NewHandler(proc)
}

func TestHealth(t *testing.T) {
	h := newTestHandler()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestProcessTransaction_Success(t *testing.T) {
	h := newTestHandler()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body, _ := json.Marshal(models.TransactionRequest{
		IdempotencyKey: "handler-test-001",
		Type:           "sale",
		Amount:         1500,
		Currency:       "USD",
		MerchantID:     "MERCH001",
		Instrument: models.InstrumentData{
			Token:       "4111111111111111",
			TokenType:   "vault",
			ExpiryMonth: 12,
			ExpiryYear:  2030,
			Brand:       "visa",
			Last4:       "1111",
		},
	})

	req := httptest.NewRequest("POST", "/v1/transactions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp models.TransactionResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "approved" {
		t.Errorf("expected approved, got %s", resp.Status)
	}
}

func TestProcessTransaction_Idempotency(t *testing.T) {
	h := newTestHandler()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body, _ := json.Marshal(models.TransactionRequest{
		IdempotencyKey: "idem-test-001",
		Type:           "sale",
		Amount:         999,
		Currency:       "USD",
		MerchantID:     "MERCH001",
		Instrument: models.InstrumentData{
			Token:       "4111111111111111",
			TokenType:   "vault",
			ExpiryMonth: 12,
			ExpiryYear:  2030,
		},
	})

	// First request.
	req1 := httptest.NewRequest("POST", "/v1/transactions", bytes.NewReader(body))
	w1 := httptest.NewRecorder()
	mux.ServeHTTP(w1, req1)

	// Second request — same idempotency key.
	req2 := httptest.NewRequest("POST", "/v1/transactions", bytes.NewReader(body))
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, req2)

	// Second should return 200 (cached), not 201.
	if w2.Code != http.StatusOK {
		t.Errorf("expected 200 for idempotent retry, got %d", w2.Code)
	}
}

func TestProcessTransaction_ValidationError(t *testing.T) {
	h := newTestHandler()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body, _ := json.Marshal(models.TransactionRequest{
		Type:   "sale",
		Amount: -100,
	})

	req := httptest.NewRequest("POST", "/v1/transactions", bytes.NewReader(body))
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422, got %d", w.Code)
	}
}
