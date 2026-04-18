package processor

import (
	"context"
	"fmt"

	moov "github.com/moov-io/iso8583"

	"github.com/mzkhan/gopay/internal/iso8583"
	"github.com/mzkhan/gopay/pkg/isofields"
)

// SimulatorSender is a stub acquirer that always approves. For dev/testing.
type SimulatorSender struct{}

func NewSimulatorSender() *SimulatorSender {
	return &SimulatorSender{}
}

func (s *SimulatorSender) Send(_ context.Context, data []byte) ([]byte, error) {
	// Parse the request to extract fields we need to echo back.
	inMsg := moov.NewMessage(isofields.Spec())
	if err := inMsg.Unpack(data); err != nil {
		return nil, fmt.Errorf("simulator: unpack request: %w", err)
	}

	var req iso8583.AuthRequest
	if err := inMsg.Unmarshal(&req); err != nil {
		return nil, fmt.Errorf("simulator: unmarshal request: %w", err)
	}

	// Build a response mirroring key fields.
	resp := &iso8583.AuthResponse{
		MTI:            "0110",
		PAN:            req.PAN,
		ProcessingCode: req.ProcessingCode,
		Amount:         req.Amount,
		STAN:           req.STAN,
		RRN:            req.RRN,
		AuthCode:       "SIM001",
		ResponseCode:   "00", // always approve
		TerminalID:     req.TerminalID,
		MerchantID:     req.MerchantID,
		NetworkRef:     "SIM" + req.STAN,
	}

	outMsg := moov.NewMessage(isofields.Spec())
	if err := outMsg.Marshal(resp); err != nil {
		return nil, fmt.Errorf("simulator: marshal response: %w", err)
	}

	packed, err := outMsg.Pack()
	if err != nil {
		return nil, fmt.Errorf("simulator: pack response: %w", err)
	}
	return packed, nil
}
