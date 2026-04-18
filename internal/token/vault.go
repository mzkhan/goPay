// Package token handles detokenization via third-party vaults (TokenEx, VGS).
// For network tokens, no detokenization is needed — the token IS the PAN substitute.
package token

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Vault detokenizes vault-stored tokens into real PANs for ISO-8583 submission.
type Vault interface {
	// Detokenize resolves a vault token to the original PAN.
	// Must only be called for token_type="vault". Network tokens bypass this.
	Detokenize(ctx context.Context, token string) (string, error)
}

// TokenExVault implements Vault for TokenEx.
type TokenExVault struct {
	baseURL   string
	tokenexID string
	apiKey    string
	client    *http.Client
}

func NewTokenExVault(baseURL, tokenexID, apiKey string) *TokenExVault {
	return &TokenExVault{
		baseURL:   baseURL,
		tokenexID: tokenexID,
		apiKey:    apiKey,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

type tokenExRequest struct {
	TokenExID string `json:"TokenExID"`
	Token     string `json:"Token"`
}

type tokenExResponse struct {
	Value        string `json:"Value"`
	Success      bool   `json:"Success"`
	Error        string `json:"Error"`
	ReferenceNum string `json:"ReferenceNumber"`
}

func (v *TokenExVault) Detokenize(ctx context.Context, token string) (string, error) {
	body, err := json.Marshal(tokenExRequest{
		TokenExID: v.tokenexID,
		Token:     token,
	})
	if err != nil {
		return "", fmt.Errorf("marshal tokenex request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.baseURL+"/Detokenize", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create tokenex request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("TX-TokenExID", v.tokenexID)
	req.Header.Set("TX-APIKey", v.apiKey)

	resp, err := v.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("tokenex request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read tokenex response: %w", err)
	}

	var result tokenExResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal tokenex response: %w", err)
	}

	if !result.Success {
		return "", fmt.Errorf("tokenex detokenization failed: %s", result.Error)
	}

	return result.Value, nil
}

// VGSVault implements Vault for Very Good Security.
type VGSVault struct {
	baseURL  string
	username string
	password string
	client   *http.Client
}

func NewVGSVault(baseURL, username, password string) *VGSVault {
	return &VGSVault{
		baseURL:  baseURL,
		username: username,
		password: password,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

type vgsRevealRequest struct {
	Data []string `json:"data"`
}

type vgsRevealResponse struct {
	Data []struct {
		Value string `json:"value"`
	} `json:"data"`
}

func (v *VGSVault) Detokenize(ctx context.Context, token string) (string, error) {
	body, err := json.Marshal(vgsRevealRequest{Data: []string{token}})
	if err != nil {
		return "", fmt.Errorf("marshal vgs request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.baseURL+"/aliases/reveal", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create vgs request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(v.username, v.password)

	resp, err := v.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("vgs request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read vgs response: %w", err)
	}

	var result vgsRevealResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("unmarshal vgs response: %w", err)
	}

	if len(result.Data) == 0 || result.Data[0].Value == "" {
		return "", fmt.Errorf("vgs returned no data for token")
	}

	return result.Data[0].Value, nil
}

// StubVault returns the token as-is. For development/testing only.
type StubVault struct{}

func (v *StubVault) Detokenize(_ context.Context, token string) (string, error) {
	return token, nil
}
