# CLAUDE.md

## Project Overview

**goPay** — a lightweight Go payment gateway that accepts JSON transaction requests with tokenized payment data and communicates with card networks (Visa, Mastercard) via ISO-8583 messaging.

### Architecture

- Pure Go, stdlib `net/http` (no frameworks)
- ISO-8583 message building/parsing via [moov-io/iso8583](https://github.com/moov-io/iso8583)
- Token detokenization via third-party vaults (TokenEx, VGS)
- Network tokens (Visa Token Service, MDES) passed directly as DPAN with cryptogram

### Token Flow

1. **Vault tokens** (`token_type: "vault"`): goPay calls TokenEx/VGS to detokenize → gets real PAN → builds ISO-8583 with PAN in field 2
2. **Network tokens** (`token_type: "network"`): Token IS the DPAN — sent directly in field 2, cryptogram goes in field 127

## Build & Run

```bash
go build ./...                           # compile
go test ./... -v                         # run tests
go run ./cmd/gopay                       # start server (port 8080)
GOPAY_API_KEY=mykey go run ./cmd/gopay   # with custom API key
```

## API

```bash
# Health check
curl http://localhost:8080/health

# Process transaction (vault token)
curl -X POST http://localhost:8080/v1/transactions \
  -H "X-API-Key: dev-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "idempotency_key": "txn-001",
    "type": "sale",
    "amount": 2500,
    "currency": "USD",
    "merchant_id": "MERCHANT01",
    "instrument": {
      "token": "tok_4111111111111111",
      "token_type": "vault",
      "expiry_month": 12,
      "expiry_year": 2030,
      "brand": "visa",
      "last4": "1111"
    }
  }'
```

## Project Structure

```
cmd/gopay/              # Entry point
internal/
  api/                  # HTTP handlers, middleware (auth, logging)
  config/               # Environment-based configuration
  iso8583/              # ISO-8583 message building/parsing (moov-io)
  models/               # Request/response types
  processor/            # Payment orchestration (detokenize → ISO → send → parse)
  token/                # Vault integrations (TokenEx, VGS, stub)
pkg/isofields/          # ISO-8583 field spec definition
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GOPAY_PORT` | `8080` | HTTP listen port |
| `GOPAY_API_KEY` | `dev-api-key` | API key for X-API-Key auth |
| `TOKEN_VAULT_PROVIDER` | `tokenex` | Vault provider: tokenex, vgs |
| `TOKEN_VAULT_URL` | `https://test-api.tokenex.com` | Vault base URL |
| `TOKEN_VAULT_API_KEY` | (empty) | Vault API key (empty = stub mode) |
| `TOKENEX_ID` | (empty) | TokenEx account ID |
