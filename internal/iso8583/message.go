// Package iso8583 builds and parses ISO-8583 messages for card network communication.
package iso8583

import (
	"fmt"
	"time"

	moov "github.com/moov-io/iso8583"

	"github.com/mzkhan/gopay/internal/models"
	"github.com/mzkhan/gopay/pkg/isofields"
)

// AuthRequest is the struct-tag mapping for an 0100 authorization request.
type AuthRequest struct {
	MTI            string `iso8583:"0"`
	PAN            string `iso8583:"2"`
	ProcessingCode string `iso8583:"3"`
	Amount         string `iso8583:"4"`
	TransmissionDT string `iso8583:"7"`
	STAN           string `iso8583:"11"`
	LocalTime      string `iso8583:"12"`
	LocalDate      string `iso8583:"13"`
	Expiry         string `iso8583:"14"`
	POSEntryMode   string `iso8583:"22"`
	POSCondCode    string `iso8583:"25"`
	RRN            string `iso8583:"37"`
	TerminalID     string `iso8583:"41"`
	MerchantID     string `iso8583:"42"`
	CurrencyCode   string `iso8583:"49"`
	NetworkToken   string `iso8583:"127"` // cryptogram + token data for network tokens
}

// AuthResponse is the struct-tag mapping for an 0110 authorization response.
type AuthResponse struct {
	MTI            string `iso8583:"0"`
	PAN            string `iso8583:"2"`
	ProcessingCode string `iso8583:"3"`
	Amount         string `iso8583:"4"`
	STAN           string `iso8583:"11"`
	RRN            string `iso8583:"37"`
	AuthCode       string `iso8583:"38"`
	ResponseCode   string `iso8583:"39"`
	TerminalID     string `iso8583:"41"`
	MerchantID     string `iso8583:"42"`
	NetworkRef     string `iso8583:"63"`
}

var spec = isofields.Spec()

// BuildAuthRequest creates a packed ISO-8583 0100 message from a transaction request.
// The PAN comes from detokenization (vault tokens) or is the network token itself.
func BuildAuthRequest(req *models.TransactionRequest, pan string, stan string, rrn string) ([]byte, error) {
	now := time.Now().UTC()

	procCode := processingCode(req.Type)
	mti := mtiForType(req.Type)
	expiry := fmt.Sprintf("%02d%02d", req.Instrument.ExpiryYear%100, req.Instrument.ExpiryMonth)

	isoReq := &AuthRequest{
		MTI:            mti,
		PAN:            pan,
		ProcessingCode: procCode,
		Amount:         fmt.Sprintf("%012d", req.Amount),
		TransmissionDT: now.Format("0102150405"),
		STAN:           stan,
		LocalTime:      now.Format("150405"),
		LocalDate:      now.Format("0102"),
		Expiry:         expiry,
		POSEntryMode:   posEntryMode(req.Instrument.TokenType),
		POSCondCode:    "59", // e-commerce
		RRN:            rrn,
		TerminalID:     truncOrPad(req.MerchantID, 8),
		MerchantID:     truncOrPad(req.MerchantID, 15),
		CurrencyCode:   currencyNumeric(req.Currency),
	}

	// For network tokens, pack cryptogram into field 127.
	if req.Instrument.TokenType == "network" && req.Instrument.Cryptogram != "" {
		isoReq.NetworkToken = fmt.Sprintf("%s|%s", req.Instrument.Cryptogram, req.Instrument.ECI)
	}

	msg := moov.NewMessage(spec)
	if err := msg.Marshal(isoReq); err != nil {
		return nil, fmt.Errorf("marshal iso request: %w", err)
	}

	packed, err := msg.Pack()
	if err != nil {
		return nil, fmt.Errorf("pack iso message: %w", err)
	}
	return packed, nil
}

// ParseAuthResponse unpacks a raw ISO-8583 0110 response.
func ParseAuthResponse(data []byte) (*AuthResponse, error) {
	msg := moov.NewMessage(spec)
	if err := msg.Unpack(data); err != nil {
		return nil, fmt.Errorf("unpack iso response: %w", err)
	}

	var resp AuthResponse
	if err := msg.Unmarshal(&resp); err != nil {
		return nil, fmt.Errorf("unmarshal iso response: %w", err)
	}
	return &resp, nil
}

func mtiForType(txnType string) string {
	switch txnType {
	case "authorization", "sale":
		return "0100"
	case "capture":
		return "0220"
	case "void":
		return "0420"
	case "refund":
		return "0100" // refund uses 0100 with different processing code
	default:
		return "0100"
	}
}

func processingCode(txnType string) string {
	switch txnType {
	case "authorization", "sale":
		return "000000"
	case "refund":
		return "200000"
	case "void":
		return "020000"
	case "capture":
		return "000000"
	default:
		return "000000"
	}
}

// posEntryMode returns POS entry mode code.
// Network tokens use 010 (credential-on-file), vault tokens use 010 as well (e-commerce/keyed).
func posEntryMode(tokenType string) string {
	if tokenType == "network" {
		return "010" // credential on file / network token
	}
	return "010" // e-commerce keyed
}

func currencyNumeric(code string) string {
	currencies := map[string]string{
		"USD": "840", "EUR": "978", "GBP": "826", "CAD": "124",
		"AUD": "036", "JPY": "392", "INR": "356", "PKR": "586",
	}
	if n, ok := currencies[code]; ok {
		return n
	}
	return "840" // default USD
}

func truncOrPad(s string, length int) string {
	if len(s) >= length {
		return s[:length]
	}
	for len(s) < length {
		s += " "
	}
	return s
}
