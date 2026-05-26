# deepenc

A privacy-preserving, multi-model AI chat platform. Conversations are encrypted with keys that are only released to a hardware-attested confidential VM (AMD SEV-SNP), so operators of the service cannot read user messages.

The frontend lets users chat with OpenAI (ChatGPT), Anthropic (Claude), and Google (Gemini) models from a single interface, switch models mid-thread, and compare responses. The backend stores only ciphertext.

---

## Table of contents

- [What this repo does](#what-this-repo-does)
- [Architecture](#architecture)
- [AMD SEV-SNP attestation](#amd-sev-snp-attestation)
- [Encryption model](#encryption-model)
- [Repository layout](#repository-layout)
- [Running locally](#running-locally)
- [Environment variables](#environment-variables)
- [API surface](#api-surface)
- [License](#license)

---

## What this repo does

User-visible features:

- Unified chat UI for ChatGPT, Claude, and Gemini.
- Multi-thread conversations with rename, delete, and search.
- Model switching within a single thread.
- Optional Retrieval-Augmented Generation (RAG) over the user's own message history via a Qdrant vector store.
- Free anonymous trial; paid tiers (Plus / Pro / Pro Plus) via Stripe.
- Per-user usage analytics and quota enforcement.
- Markdown rendering with code highlighting, KaTeX math, and Mermaid diagrams.

Security posture:

- Message content is encrypted with AES-256-GCM before it is stored.
- The wrapping key is held in Azure Key Vault and is released only against a valid Microsoft Azure Attestation (MAA) token that asserts the server is running inside an AMD SEV-SNP confidential VM with a measurement matching a policy.
- The vector store holds embeddings only, never plaintext.

---

## Architecture

```
                 Browser (Angular SPA)
                          |
                       HTTPS
                          v
                       nginx  (TLS termination, static + reverse proxy)
                          |
                          v
       +------------------+------------------+
       |   Go HTTP server (deepenc-server)   |
       |   running inside AMD SEV-SNP VM     |
       +------------------+------------------+
          |        |         |         |
          v        v         v         v
   Cosmos DB    Key Vault   Qdrant   LLM providers
   (ciphertext) (wrapped    (embeds) (OpenAI /
                key, MAA-             Anthropic /
                gated)                Gemini)
                ^
                |
                | MAA attestation token (SEV-SNP claims)
                |
        Microsoft Azure Attestation
```

**Backend** — Go 1.24, standard `net/http` (no web framework). Notable packages:

- `backend/cmd/server/main.go` — HTTP routing and request handlers.
- `backend/cmd/server/encryption.go` — message encryption/decryption.
- `backend/cmd/server/cosmos_store.go`, `multi_*_store.go` — Cosmos DB persistence.
- `backend/cmd/server/vector_store.go` — Qdrant RAG integration.
- `backend/cmd/server/stripe_service.go` — Stripe billing & subscriptions.
- `backend/cmd/server/usage_service.go`, `token_calculator.go` — tier-based quota enforcement.
- `backend/cmd/server/auth_middleware.go` — Firebase ID token verification.
- `backend/internal/attestation/` — SEV-SNP attestation client, MAA token validation, Key Vault release policy. See [AMD SEV-SNP attestation](#amd-sev-snp-attestation).

**Frontend** — Angular 20, standalone components. Built with the Angular CLI. Key libraries: `firebase` (auth), `@stripe/stripe-js` (checkout), `marked` + `ngx-markdown`, `highlight.js`, `katex`, `mermaid`, `chart.js`.

**Data stores**:

- **Azure Cosmos DB** (SQL API) — encrypted messages, threads, users, subscription state, webhook event log.
- **Azure Key Vault** — wrapped master key, releasable only against a valid MAA attestation token.
- **Qdrant** (optional) — embedding vectors for RAG. Embeddings are generated from plaintext inside the TEE; only vectors leave the enclave.

**Third-party APIs**: OpenAI, Anthropic, Google Gemini, Stripe, Firebase Auth, Azure (Key Vault, Cosmos DB, Attestation).

---

## AMD SEV-SNP attestation

AMD SEV-SNP (Secure Encrypted Virtualization — Secure Nested Paging) is a CPU feature that encrypts and integrity-protects a guest VM's memory from the hypervisor and host OS. A SEV-SNP guest can request a hardware-signed *attestation report* that includes a measurement of the launched code, which a remote verifier can use to gate the release of secrets.

This repo uses the Azure hosted variant of SEV-SNP and the Microsoft Azure Attestation (MAA) service.

### Attestation flow

1. **Boot.** The server starts inside an Azure Confidential VM with SEV-SNP enabled. The host exposes the AMD-SP report endpoint and a TPM (`/dev/tpmrm0`).
2. **Acquire a SEV-SNP report.** The Go server calls into a CGO wrapper (`backend/internal/attestation/cgo/`) which invokes Azure's `libazguestattestation1`. The wrapper produces a SEV-SNP report bound to a runtime nonce and submits it to the MAA endpoint.
3. **Receive an MAA token.** MAA validates the SEV-SNP report against AMD's signing chain and returns a JWT (the MAA token) containing claims such as `x-ms-isolation-tee: sevsnpvm`, the launch measurement, and the report data.
4. **Validate the token locally.** `token_validator.go` fetches MAA's JWKS, verifies the JWT signature, checks `iss`/`exp`/`nbf`, and extracts the SEV-SNP claims (`claims.go`).
5. **Release the wrapped key.** The MAA token is presented to Azure Key Vault via the `x-ms-attestation-token` header. Key Vault evaluates the [Secure Key Release](https://learn.microsoft.com/en-us/azure/key-vault/keys/policy-grammar) policy installed on the key, which requires `x-ms-isolation-tee == "sevsnpvm"` and a launch measurement matching the value configured for this deployment (`MAA_POLICY_MEASUREMENT`). If the policy is satisfied, Key Vault releases the wrapping key into the enclave.
6. **Encrypt/decrypt messages.** The wrapping key stays in process memory protected by SEV-SNP. Per-message AES-256-GCM data keys are wrapped under it before being persisted.

### Why this matters

The threat model assumes a curious or compromised operator: someone with root on the host, hypervisor access, physical access to the datacenter, or stolen Cosmos DB credentials. Against that adversary:

- Memory is encrypted by the CPU; the hypervisor sees only ciphertext.
- The key needed to decrypt stored messages is released only to a VM running code whose measurement matches what the user (or auditor) can independently verify.
- Tampering with the deployed binary changes the measurement and breaks key release.
- The database holds ciphertext; the vector store holds only embeddings.

### Files

- `backend/internal/attestation/manager.go` — orchestration.
- `backend/internal/attestation/client.go` — CGO bridge to `libazguestattestation1`.
- `backend/internal/attestation/client_stub.go` — pure-Go stub for non-Linux / non-CGO builds, used in local development.
- `backend/internal/attestation/token_validator.go` — JWKS fetch and JWT validation.
- `backend/internal/attestation/claims.go` — typed view of SEV-SNP claims.
- `backend/internal/attestation/keyvault_policy.go` — helpers for emitting the SKR policy that the Key Vault key is created with.
- `backend/internal/attestation/cgo/` — C++ wrapper around the Azure guest attestation library.

### Development without an SEV-SNP host

The attestation client has a build-tag-gated stub (`client_stub.go`) so the server compiles and runs on macOS / Windows / non-CGO Linux. In that mode, encryption falls back to a locally-held key (or plaintext if no key is configured) — useful for development, **not safe for production**.

---

## Encryption model

- Per-message AES-256-GCM with a random 12-byte nonce and a random 256-bit data key.
- The data key is wrapped under a master key released from Key Vault as described above.
- Cosmos DB stores `{ciphertext, wrapped_data_key, nonce, auth_tag, algorithm}`. No plaintext.
- For RAG: plaintext is embedded inside the enclave (using OpenAI's embedding API), the resulting vector is written to Qdrant tagged with the message ID, and the plaintext is discarded. Search returns IDs; the corresponding ciphertexts are fetched from Cosmos and decrypted in-enclave before being added to the LLM context.

See `backend/cmd/server/encryption.go` and `backend/cmd/server/vector_store.go`.

---

## Repository layout

```
backend/
  cmd/server/                 Go HTTP server (entry point: main.go)
    .env.example              Backend env-var template
  internal/attestation/       SEV-SNP attestation + MAA validation
    cgo/                      C++ wrapper around libazguestattestation1
frontend/
  src/app/                    Angular components, services, routes
  src/environments/           Per-environment frontend config
  proxy.conf*.json            Dev-server proxy to the backend
  nginx.conf                  Frontend's production nginx config
nginx.conf                    Reference top-level nginx config
LICENSE                       BSD 3-Clause
```

---

## Running locally

### Prerequisites

- Go 1.24+
- Node.js 20.19+ and npm
- An Azure Cosmos DB account (or the Cosmos emulator)
- A Firebase project with Authentication enabled and a service-account JSON
- API keys for whichever LLM providers you want to enable (OpenAI / Anthropic / Gemini)
- (Optional) A Stripe test account if you want to exercise billing
- (Optional) A running Qdrant instance for RAG

You do **not** need an SEV-SNP host to run locally — the attestation stub kicks in automatically on macOS / non-CGO builds.

### Backend

```bash
cd backend/cmd/server
cp .env.example .env
# edit .env with your config (see "Environment variables" below)
go run .
```

The server listens on `:8080` by default.

### Frontend

```bash
cd frontend
npm install
npm run start:local
```

The dev server runs on `http://localhost:4200` and proxies `/api/*` to the backend (see `frontend/proxy.conf.local.json`).

---

## Environment variables

See `backend/cmd/server/.env.example` for the full list. Key groups:

**Server**

- `PORT` (default `8080`)
- `ENVIRONMENT` (`development` | `production`)
- `ALLOWED_ORIGINS` — comma-separated CORS origins
- `SESSION_SECRET` — 256-bit hex
- `MAX_REQUEST_SIZE_MB`, `RATE_LIMIT_ANON_REQUESTS_PER_MINUTE`

**Cosmos DB**

- `COSMOS_ENDPOINT`, `COSMOS_KEY`, `COSMOS_DATABASE_NAME`, `COSMOS_CONTAINER_NAME`
- `COSMOS_EMULATOR_INSECURE=true` for local emulator

**Firebase**

- `FIREBASE_PROJECT_ID`, `FIREBASE_AUTH_DOMAIN`, `FIREBASE_APP_ID`, etc.
- `FIREBASE_SERVICE_ACCOUNT_PATH` or `FIREBASE_SERVICE_ACCOUNT_JSON`

**LLM providers** (each optional; the corresponding models are disabled if its key is unset)

- `OPENAI_API_KEY`, `OPENAI_MODELS`
- `ANTHROPIC_API_KEY`, `ANTHROPIC_MODELS`
- `GEMINI_API_KEY`, `GEMINI_MODELS`

**Stripe**

- `STRIPE_PUBLISHABLE_KEY`, `STRIPE_SECRET_KEY`, `STRIPE_WEBHOOK_SECRET`
- Per-tier product/price IDs: `STRIPE_PLUS_PRICE_ID`, `STRIPE_PRO_PRICE_ID`, `STRIPE_PRO_PLUS_PRICE_ID`, etc.

**RAG / vector store** (optional)

- `QDRANT_URL`, `QDRANT_SCORE_THRESHOLD`, `EMBEDDING_MODEL`

**Attestation & Key Vault** (required for production)

- `AZURE_KEY_VAULT_URL`
- `AZURE_KEY_VAULT_SECRET_NAME` (or key name)
- `MAA_ENDPOINT` (e.g. `https://sharedeus2.eus2.attest.azure.net`)
- `MAA_POLICY_MEASUREMENT` — expected SEV-SNP launch measurement

---

## API surface

A non-exhaustive list of HTTP routes exposed by the backend (`backend/cmd/server/main.go`):

| Method | Path | Auth | Purpose |
|---|---|---|---|
| GET | `/api/healthz`, `/api/health` | none | Liveness probes |
| GET | `/api/config/firebase`, `/api/config/stripe` | none | Public client config |
| GET | `/api/models` | optional | List available models for this user/tier |
| GET/POST | `/api/threads`, `/api/threads/{id}` | required | Thread CRUD |
| POST | `/api/stream/contextual` | optional | Streamed chat completion with context assembly |
| POST | `/api/parse/{openai,claude,gemini,content}` | none | Markdown/code/diagram extraction |
| GET | `/api/usage/stats`, `/api/usage/analytics` | required | Quota and usage |
| GET | `/api/usage/tiers` | none | Tier definitions and pricing |
| GET/POST | `/api/subscription/*` | required | Stripe checkout, billing portal, cancel, renew |
| POST | `/api/webhooks/stripe` | signature | Stripe event ingestion |
| POST | `/api/auth/{register,profile,delete}` | required | Firebase-authenticated account management |

---

## License

BSD 3-Clause. See [LICENSE](LICENSE).
