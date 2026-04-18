# goPay

A lightweight Go payment gateway that accepts JSON transaction requests with tokenized payment data and processes them via ISO-8583 messaging for card network communication (Visa/Mastercard).

## Features

- **ISO-8583 messaging** — Builds and parses ISO-8583:1987 messages for communication with Visa Base I and Mastercard MIP acquirer hosts
- **Token vault integration** — Detokenizes payment tokens via third-party vaults (TokenEx, VGS) before sending to card networks
- **Network token support** — Accepts Visa Token Service (VTS) and Mastercard Digital Enablement Service (MDES) network tokens with cryptogram passthrough
- **Idempotency** — Duplicate requests with the same `idempotency_key` return the cached response
- **API key authentication** — All transaction endpoints require an `X-API-Key` header
- **Simulator mode** — Built-in acquirer simulator that approves all transactions for development and testing
- **Structured logging** — JSON-formatted request/response logging via `slog`
- **Graceful shutdown** — Handles SIGINT/SIGTERM for clean connection draining

## How It Works

```
                         ┌─────────────┐
  JSON Request ──────▶   │   goPay     │
  (tokenized data)       │             │
                         │  1. Validate │
                         │  2. Detokenize (vault tokens)
                         │  3. Build ISO-8583
                         │  4. Send to acquirer
                         │  5. Parse ISO response
                         │  6. Return JSON
  JSON Response  ◀────── │             │
                         └─────────────┘
```

**Vault tokens** (`token_type: "vault"`): goPay calls TokenEx or VGS to detokenize the token into a real PAN, then places the PAN in ISO-8583 field 2.

**Network tokens** (`token_type: "network"`): The token is a Device PAN (DPAN) issued by the card network. It is sent directly in field 2 with the cryptogram in field 127. No detokenization is needed.

## Quick Start

### Prerequisites

- Go 1.21+

### Run

```bash
go run ./cmd/gopay
```

The server starts on port `8080` with the stub vault and simulator acquirer.

### Build

```bash
go build -o gopay ./cmd/gopay
./gopay
```

### Test

```bash
go test ./... -v
```

## Configuration

All configuration is via environment variables.

| Variable | Default | Description |
|----------|---------|-------------|
| `GOPAY_PORT` | `8080` | HTTP listen port |
| `GOPAY_API_KEY` | `dev-api-key` | API key for `X-API-Key` authentication |
| `TOKEN_VAULT_PROVIDER` | `tokenex` | Vault provider: `tokenex` or `vgs` |
| `TOKEN_VAULT_URL` | `https://test-api.tokenex.com` | Vault API base URL |
| `TOKEN_VAULT_API_KEY` | _(empty)_ | Vault API key. If empty, the stub vault is used |
| `TOKENEX_ID` | _(empty)_ | TokenEx account ID |

Example:

```bash
GOPAY_PORT=9090 GOPAY_API_KEY=my-secret-key go run ./cmd/gopay
```

## API Reference

### Health Check

```
GET /health
```

No authentication required.

**Request:**

```bash
curl http://localhost:8080/health
```

**Response:** `200 OK`

```json
{
  "status": "ok"
}
```

---

### Process Transaction

```
POST /v1/transactions
```

Requires `X-API-Key` header.

#### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `idempotency_key` | string | Yes | Unique key to prevent duplicate processing |
| `type` | string | Yes | `authorization`, `sale`, `capture`, `void`, or `refund` |
| `amount` | integer | Yes | Amount in minor units (e.g. `2500` = $25.00) |
| `currency` | string | Yes | 3-letter ISO currency code (e.g. `USD`) |
| `merchant_id` | string | Yes | Merchant identifier |
| `instrument` | object | Yes | Payment instrument (see below) |
| `original_ref` | string | No | Original transaction reference (for capture/void/refund) |

**Instrument object:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `token` | string | Yes | Token from vault or network token (DPAN) |
| `token_type` | string | Yes | `vault` (TokenEx/VGS) or `network` (Visa/MC network token) |
| `cryptogram` | string | For network | Token cryptogram proving validity |
| `eci` | string | For network | Electronic Commerce Indicator (e.g. `05`) |
| `expiry_month` | integer | Yes | Card expiry month (1-12) |
| `expiry_year` | integer | Yes | Card expiry year (e.g. 2030) |
| `brand` | string | No | Card brand hint: `visa`, `mastercard` |
| `last4` | string | No | Last 4 digits for display/receipt purposes |

#### Response Body

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Transaction ID |
| `idempotency_key` | string | Echoed back from request |
| `status` | string | `approved`, `declined`, or `error` |
| `type` | string | Transaction type |
| `amount` | integer | Amount in minor units |
| `currency` | string | Currency code |
| `auth_code` | string | Authorization code from the network |
| `response_code` | string | ISO response code (`00` = approved) |
| `response_message` | string | Human-readable response message |
| `network_ref` | string | Network/acquirer reference number |
| `processed_at` | string | ISO-8601 timestamp |

---

### Examples

#### Sale with Vault Token

A vault token (from TokenEx or VGS) is detokenized server-side before being sent to the card network.

```bash
curl -X POST http://localhost:8080/v1/transactions \
  -H "X-API-Key: dev-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "idempotency_key": "order-12345",
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

**Response:** `201 Created`

```json
{
  "id": "784608-260418015294",
  "idempotency_key": "order-12345",
  "status": "approved",
  "type": "sale",
  "amount": 2500,
  "currency": "USD",
  "auth_code": "SIM001",
  "response_code": "00",
  "response_message": "Approved",
  "network_ref": "260418015294",
  "processed_at": "2026-04-18T01:13:29.345409Z"
}
```

#### Authorization with Network Token

A network token (Visa VTS or Mastercard MDES) is sent directly to the network as a DPAN with a cryptogram.

```bash
curl -X POST http://localhost:8080/v1/transactions \
  -H "X-API-Key: dev-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "idempotency_key": "auth-67890",
    "type": "authorization",
    "amount": 10000,
    "currency": "USD",
    "merchant_id": "MERCHANT01",
    "instrument": {
      "token": "4900000000000001",
      "token_type": "network",
      "cryptogram": "AABBCCDD11223344",
      "eci": "05",
      "expiry_month": 6,
      "expiry_year": 2029,
      "brand": "visa",
      "last4": "0001"
    }
  }'
```

**Response:** `201 Created`

```json
{
  "id": "251936-260418010858",
  "idempotency_key": "auth-67890",
  "status": "approved",
  "type": "authorization",
  "amount": 10000,
  "currency": "USD",
  "auth_code": "SIM001",
  "response_code": "00",
  "response_message": "Approved",
  "network_ref": "260418010858",
  "processed_at": "2026-04-18T01:12:42.813509Z"
}
```

#### Idempotent Retry

Sending the same `idempotency_key` returns the cached response with `200 OK` instead of `201 Created`.

```bash
# First request — returns 201
curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:8080/v1/transactions \
  -H "X-API-Key: dev-api-key" \
  -H "Content-Type: application/json" \
  -d '{"idempotency_key":"retry-001","type":"sale","amount":500,"currency":"USD","merchant_id":"M1","instrument":{"token":"4111111111111111","token_type":"vault","expiry_month":12,"expiry_year":2030}}'
# Output: 201

# Same request again — returns 200 (cached)
curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:8080/v1/transactions \
  -H "X-API-Key: dev-api-key" \
  -H "Content-Type: application/json" \
  -d '{"idempotency_key":"retry-001","type":"sale","amount":500,"currency":"USD","merchant_id":"M1","instrument":{"token":"4111111111111111","token_type":"vault","expiry_month":12,"expiry_year":2030}}'
# Output: 200
```

#### Validation Error

```bash
curl -X POST http://localhost:8080/v1/transactions \
  -H "X-API-Key: dev-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "sale",
    "amount": -100,
    "currency": "USD"
  }'
```

**Response:** `422 Unprocessable Entity`

```json
{
  "error": "idempotency_key is required",
  "code": "VALIDATION_ERROR"
}
```

#### Unauthorized Request

```bash
curl -X POST http://localhost:8080/v1/transactions \
  -H "Content-Type: application/json" \
  -d '{}'
```

**Response:** `401 Unauthorized`

```
{"error":"unauthorized"}
```

## Transaction Types

| Type | ISO-8583 MTI | Processing Code | Description |
|------|-------------|-----------------|-------------|
| `authorization` | 0100 | 000000 | Places a hold on funds without capturing |
| `sale` | 0100 | 000000 | Authorizes and captures in one step |
| `capture` | 0220 | 000000 | Captures a previously authorized transaction |
| `void` | 0420 | 020000 | Cancels a transaction before settlement |
| `refund` | 0100 | 200000 | Returns funds to the cardholder |

## Response Codes

| Code | Message |
|------|---------|
| `00` | Approved |
| `01` | Refer to issuer |
| `05` | Do not honor |
| `12` | Invalid transaction |
| `13` | Invalid amount |
| `14` | Invalid card number |
| `51` | Insufficient funds |
| `54` | Expired card |
| `55` | Incorrect PIN |
| `61` | Exceeds withdrawal limit |
| `91` | Issuer unavailable |
| `96` | System malfunction |

## Project Structure

```
cmd/gopay/                  Entry point
internal/
  api/                      HTTP handlers, middleware (auth, logging)
  config/                   Environment-based configuration
  iso8583/                  ISO-8583 message building and parsing
  models/                   Request/response types
  processor/                Payment orchestration pipeline
  token/                    Vault integrations (TokenEx, VGS, stub)
pkg/isofields/              ISO-8583 field specification
```

## License

Proprietary.
