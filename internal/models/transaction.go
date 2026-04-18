package models

import "time"

// TransactionRequest is the JSON payload accepted from clients.
type TransactionRequest struct {
	// Idempotency key — callers must provide a unique value per attempt.
	IdempotencyKey string `json:"idempotency_key" validate:"required"`

	// Transaction type: "authorization", "sale", "capture", "void", "refund"
	Type string `json:"type" validate:"required,oneof=authorization sale capture void refund"`

	Amount    int64  `json:"amount" validate:"required,gt=0"` // minor units (cents)
	Currency  string `json:"currency" validate:"required,len=3"`
	MerchantID string `json:"merchant_id" validate:"required"`

	// Payment instrument — token-based. The actual PAN is never sent to goPay.
	Instrument InstrumentData `json:"instrument" validate:"required"`

	// Optional: original transaction reference for capture/void/refund.
	OriginalRef string `json:"original_ref,omitempty"`
}

// InstrumentData carries tokenized card data.
type InstrumentData struct {
	// Token from the vault (TokenEx, VGS, or a network token).
	Token string `json:"token" validate:"required"`

	// Token type: "vault" (TokenEx/VGS detokenize-to-PAN) or "network" (Visa/MC network token).
	TokenType string `json:"token_type" validate:"required,oneof=vault network"`

	// For network tokens: the cryptogram proving token validity.
	Cryptogram string `json:"cryptogram,omitempty"`

	// For network tokens: Electronic Commerce Indicator.
	ECI string `json:"eci,omitempty"`

	ExpiryMonth int `json:"expiry_month" validate:"required,min=1,max=12"`
	ExpiryYear  int `json:"expiry_year" validate:"required"`

	// Card brand hint — used for routing. "visa", "mastercard", etc.
	Brand string `json:"brand,omitempty"`

	// Last four digits for logging/receipt purposes only.
	Last4 string `json:"last4,omitempty"`
}

// TransactionResponse is returned to the caller.
type TransactionResponse struct {
	ID             string    `json:"id"`
	IdempotencyKey string    `json:"idempotency_key"`
	Status         string    `json:"status"` // "approved", "declined", "error"
	Type           string    `json:"type"`
	Amount         int64     `json:"amount"`
	Currency       string    `json:"currency"`
	AuthCode       string    `json:"auth_code,omitempty"`
	ResponseCode   string    `json:"response_code"`
	ResponseMsg    string    `json:"response_message"`
	NetworkRef     string    `json:"network_ref,omitempty"` // acquirer/network reference
	ProcessedAt    time.Time `json:"processed_at"`
}

// ErrorResponse is a standard error envelope.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}
