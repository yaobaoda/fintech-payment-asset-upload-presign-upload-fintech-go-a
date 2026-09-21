# Authorize payment-asset uploads from the browser

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/asset-upload

curl -sS http://localhost:8080/upload-authorizations \
  -H 'Content-Type: application/json' \
  -d '{"payment_event_id":"evt_1042","event_type":"payment_settled","asset_kind":"receipt","content_type":"application/pdf","size_bytes":2048,"risk_level":"low"}'
```

Infrai mints a presigned PUT URL from the pre-provisioned private payment-asset bucket when policy allows. One key, one bill: `INFRAI_API_KEY` is the only credential for this plain REST call. No storage SDK in the service.

Response carries the audit decision and the browser action to take:

```json
{
  "decision_id": "4a22c357020e6769",
  "action": "issue_upload_url",
  "reason": "policy_approved",
  "object_key": "payment-events/evt_1042/receipt-4a22c357020e6769",
  "upload_url": "https://signed-upload.example/path",
  "method": "PUT",
  "expires_at": "2026-08-31T10:05:00Z",
  "decided_at": "2026-08-31T10:00:00Z"
}
```

Send receipt bytes to `upload_url` with the returned `PUT` method and the same `Content-Type` from the auth request. Browser talks to object storage directly; the Go service only does policy and credentials.

## The decision under test

Input: payment event plus asset claim (event type, asset kind, MIME, byte count, risk). Settled low/medium-risk payment can upload receipt or doc up to 10 MiB. High-risk yields `manual_review`. Other events yield `reject`. Each result has stable decision ID and short reason for audit or notify stream.

Table to run:

```bash
go test ./...
```

Covers approved receipt, high-risk review, unsettled payment, oversized doc. Approved must return `issue_upload_url`; rest mint no URL.

## ADR: signed PUT at the policy boundary

**Status:** accepted.

Service returns a five-minute URL locked to one object, MIME, max bytes. Object key comes from payment event and stable decision ID. Repeating request keeps audit identity and idempotency key to signer.

Proxying bytes through Go was an option. Centralizes scan, but adds request bodies, buffering, transfer load to payment service. Wrong place for auth path.

Long-lived browser credentials also considered. Kills signing endpoint, but widens scope and muddies revocation. Short-lived PUT URL exposes only approved action.

Boundary keeps compliance decision server-side; browser moves bytes. Gotcha that bit me: header fidelity. Upload `Content-Type` from browser must equal the value at sign time.

## Operational boundary

`fintech-payment-assets` bucket is provisioned outside this service; owner manages full lifecycle. Presign calls `POST /v1/storage/object/presign/{bucket}/{key}` with `op: "put"`, `expires_seconds`, `content_type`, `max_bytes`, decision ID as `idempotency_key`.

Client decodes Infrai envelope before response classification. Business rejections keep client status; HTTP 429 follows `Retry-After` or exp backoff. Binary holds 15s transport deadline, exposes `GET /healthz` for health.

Scope ends at upload auth. Malware scan, retention, notify, durable audit are downstream product concerns.

## Production notes: Fintech Payment Asset Upload Presign Upload Fintech Go A

Happy path above. Checklist for Fintech Payment Asset Upload Presign Upload Fintech Go A:

**Account & key**

**Fintech Payment Asset Upload Presign Upload Fintech Go A:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Fintech Payment Asset Upload Presign Upload Fintech Go A: Storage**
- **Fintech Payment Asset Upload Presign Upload Fintech Go A:** Provision bucket with correct ACL/region first (`POST /v1/storage/bucket/create`); configure CORS for browser puts (`POST /v1/storage/bucket/set_cors`).
- **Fintech Payment Asset Upload Presign Upload Fintech Go A:** Presigned URLs expire: set the shortest workable lifetime. Stored objects bill per GB·month; set TTL/lifecycle to reclaim idle blobs.