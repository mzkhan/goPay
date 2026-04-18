// Package processor orchestrates payment processing: detokenize → build ISO → send → parse response.
package processor

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/mzkhan/gopay/internal/iso8583"
	"github.com/mzkhan/gopay/internal/models"
	"github.com/mzkhan/gopay/internal/token"
)

// Sender sends packed ISO-8583 bytes to an acquirer and returns the raw response.
type Sender interface {
	Send(ctx context.Context, data []byte) ([]byte, error)
}

// Processor handles the full transaction lifecycle.
type Processor struct {
	vault  token.Vault
	sender Sender
	stan   atomic.Uint64
}

func New(vault token.Vault, sender Sender) *Processor {
	p := &Processor{
		vault:  vault,
		sender: sender,
	}
	// Seed STAN counter from random to avoid collisions across restarts.
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	p.stan.Store(uint64(n.Int64()) + 1)
	return p
}

// Process handles an inbound transaction request end-to-end.
func (p *Processor) Process(ctx context.Context, req *models.TransactionRequest) (*models.TransactionResponse, error) {
	logger := slog.With("idempotency_key", req.IdempotencyKey, "type", req.Type)

	// 1. Resolve PAN — either detokenize vault token or use network token directly.
	pan, err := p.resolvePAN(ctx, &req.Instrument)
	if err != nil {
		logger.Error("detokenization failed", "error", err)
		return errorResponse(req, "TOKEN_ERROR", "Failed to resolve payment token"), nil
	}

	// 2. Generate STAN and RRN.
	stan := fmt.Sprintf("%06d", p.nextSTAN())
	rrn := generateRRN()

	// 3. Build ISO-8583 message.
	packed, err := iso8583.BuildAuthRequest(req, pan, stan, rrn)
	if err != nil {
		logger.Error("ISO build failed", "error", err)
		return errorResponse(req, "ISO_ERROR", "Failed to build ISO message"), nil
	}

	// 4. Send to acquirer/network.
	rawResp, err := p.sender.Send(ctx, packed)
	if err != nil {
		logger.Error("acquirer send failed", "error", err)
		return errorResponse(req, "NETWORK_ERROR", "Failed to reach card network"), nil
	}

	// 5. Parse ISO-8583 response.
	isoResp, err := iso8583.ParseAuthResponse(rawResp)
	if err != nil {
		logger.Error("ISO parse failed", "error", err)
		return errorResponse(req, "PARSE_ERROR", "Failed to parse network response"), nil
	}

	// 6. Map to TransactionResponse.
	status := "declined"
	if isoResp.ResponseCode == "00" {
		status = "approved"
	}

	resp := &models.TransactionResponse{
		ID:             fmt.Sprintf("%s-%s", stan, rrn),
		IdempotencyKey: req.IdempotencyKey,
		Status:         status,
		Type:           req.Type,
		Amount:         req.Amount,
		Currency:       req.Currency,
		AuthCode:       isoResp.AuthCode,
		ResponseCode:   isoResp.ResponseCode,
		ResponseMsg:    responseMessage(isoResp.ResponseCode),
		NetworkRef:     isoResp.RRN,
		ProcessedAt:    time.Now().UTC(),
	}

	logger.Info("transaction processed", "status", status, "response_code", isoResp.ResponseCode)
	return resp, nil
}

func (p *Processor) resolvePAN(ctx context.Context, inst *models.InstrumentData) (string, error) {
	switch inst.TokenType {
	case "network":
		// Network tokens ARE the PAN substitute — sent directly in field 2.
		return inst.Token, nil
	case "vault":
		// Vault tokens must be detokenized to get the real PAN.
		return p.vault.Detokenize(ctx, inst.Token)
	default:
		return "", fmt.Errorf("unknown token type: %s", inst.TokenType)
	}
}

func (p *Processor) nextSTAN() uint64 {
	v := p.stan.Add(1)
	return v % 1000000 // keep within 6 digits
}

func generateRRN() string {
	now := time.Now().UTC()
	n, _ := rand.Int(rand.Reader, big.NewInt(9999))
	return fmt.Sprintf("%s%04d", now.Format("06010215"), n.Int64())
}

func errorResponse(req *models.TransactionRequest, code, msg string) *models.TransactionResponse {
	return &models.TransactionResponse{
		IdempotencyKey: req.IdempotencyKey,
		Status:         "error",
		Type:           req.Type,
		Amount:         req.Amount,
		Currency:       req.Currency,
		ResponseCode:   code,
		ResponseMsg:    msg,
		ProcessedAt:    time.Now().UTC(),
	}
}

func responseMessage(code string) string {
	messages := map[string]string{
		"00": "Approved",
		"01": "Refer to issuer",
		"05": "Do not honor",
		"12": "Invalid transaction",
		"13": "Invalid amount",
		"14": "Invalid card number",
		"51": "Insufficient funds",
		"54": "Expired card",
		"55": "Incorrect PIN",
		"61": "Exceeds withdrawal limit",
		"91": "Issuer unavailable",
		"96": "System malfunction",
	}
	if m, ok := messages[code]; ok {
		return m
	}
	return "Declined"
}
