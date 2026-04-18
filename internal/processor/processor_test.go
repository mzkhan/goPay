package processor_test

import (
	"context"
	"testing"

	"github.com/mzkhan/gopay/internal/models"
	"github.com/mzkhan/gopay/internal/processor"
	"github.com/mzkhan/gopay/internal/token"
)

func TestProcess_VaultToken_Approved(t *testing.T) {
	vault := &token.StubVault{}
	sender := processor.NewSimulatorSender()
	proc := processor.New(vault, sender)

	req := &models.TransactionRequest{
		IdempotencyKey: "test-001",
		Type:           "sale",
		Amount:         2500,
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
	}

	resp, err := proc.Process(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "approved" {
		t.Errorf("expected approved, got %s", resp.Status)
	}
	if resp.ResponseCode != "00" {
		t.Errorf("expected response code 00, got %s", resp.ResponseCode)
	}
	if resp.AuthCode == "" {
		t.Error("expected non-empty auth code")
	}
	if resp.Amount != 2500 {
		t.Errorf("expected amount 2500, got %d", resp.Amount)
	}
}

func TestProcess_NetworkToken_Approved(t *testing.T) {
	vault := &token.StubVault{} // won't be called for network tokens
	sender := processor.NewSimulatorSender()
	proc := processor.New(vault, sender)

	req := &models.TransactionRequest{
		IdempotencyKey: "test-002",
		Type:           "authorization",
		Amount:         5000,
		Currency:       "USD",
		MerchantID:     "MERCH002",
		Instrument: models.InstrumentData{
			Token:       "4900000000000001", // network token (DPAN)
			TokenType:   "network",
			Cryptogram:  "AABBCCDD11223344",
			ECI:         "05",
			ExpiryMonth: 6,
			ExpiryYear:  2029,
			Brand:       "visa",
			Last4:       "0001",
		},
	}

	resp, err := proc.Process(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "approved" {
		t.Errorf("expected approved, got %s", resp.Status)
	}
	if resp.IdempotencyKey != "test-002" {
		t.Errorf("expected idempotency key test-002, got %s", resp.IdempotencyKey)
	}
}

func TestProcess_InvalidTokenType(t *testing.T) {
	vault := &token.StubVault{}
	sender := processor.NewSimulatorSender()
	proc := processor.New(vault, sender)

	req := &models.TransactionRequest{
		IdempotencyKey: "test-003",
		Type:           "sale",
		Amount:         1000,
		Currency:       "USD",
		MerchantID:     "MERCH003",
		Instrument: models.InstrumentData{
			Token:       "some-token",
			TokenType:   "unknown",
			ExpiryMonth: 1,
			ExpiryYear:  2028,
		},
	}

	resp, err := proc.Process(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "error" {
		t.Errorf("expected error status, got %s", resp.Status)
	}
	if resp.ResponseCode != "TOKEN_ERROR" {
		t.Errorf("expected TOKEN_ERROR, got %s", resp.ResponseCode)
	}
}
