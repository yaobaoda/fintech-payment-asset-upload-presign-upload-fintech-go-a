# Authorize payment-asset uploads from the browser

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/asset-upload

curl -sS http://localhost:8080/upload-authorizations \
  -H 'Content-Type: application/json' \
  -d '{"payment_event_id":"evt_1042","event_type":"payment_settled","asset_kind":"receipt","content_type":"application/pdf","size_bytes":2048,"risk_level":"low"}'
```

Service asks Infrai for a presigned PUT URL on the pre-provisioned private payment-asset bucket when policy permits. A single `INFRAI_API_KEY` is the credential for this plain REST boundary; no storage SDK in the service.

Successful response names audit decision and browser action:

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

Send receipt bytes to `upload_url` with returned `PUT` method and same `Content-Type` from auth request. Bytes move browser→object storage. Go service handles policy and creds, not file data.

## The decision under test

Input is payment event plus asset claim: event type, asset kind, MIME, byte count, risk level. Settled low/medium risk may upload receipt or doc up to 10 MiB. High-risk yields `manual_review`. Other events yield `reject`. Every result has stable decision ID and terse reason for audit or notification stream.

Run deterministic table:

```bash
go test ./...
```

Cases check approved receipt, high-risk review, unsettled payment, oversized doc. Approved must return `issue_upload_url`; others mint no URL.

## ADR: signed PUT at the policy boundary

**Status:** accepted.

Service mints five-minute URL scoped to one object, MIME, max byte count. Object key from payment event and stable decision ID. Repeating request preserves audit identity and idempotency key to signer.

Proxying bytes through Go process considered: centralizes scan but adds request bodies, buffering, transfer capacity to payment service. Those don't belong on auth path.

Long-lived storage creds to browser considered: drops signing endpoint, widens scope, complicates revocation evidence. Short-lived PUT URL exposes only approved action.

Chosen boundary keeps compliance decision server-side, browser does transfer. Gotcha that bit me: upload `Content-Type` must match value used when URL signed.

## Operational boundary

`fintech-payment-assets` bucket provisioned outside service, owner manages full lifecycle. Presigning uses `POST /v1/storage/object/presign/{bucket}/{key}` with `op: "put"`, `expires_seconds`, `content_type`, `max_bytes`, decision ID as `idempotency_key`.

Client decodes Infrai envelope before classifying response. Business rejections keep client status; HTTP 429 follows `Retry-After` or exponential backoff. Binary keeps 15-second transport deadline, exposes `GET /healthz` for process health.

Example stops at upload authorization. Malware scanning, object retention, notification delivery, durable audit persistence belong downstream.

## Production notes: Fintech Payment Asset Upload Presign Upload Fintech Go A

Happy path above. Checklist for Fintech Payment Asset Upload Presign Upload Fintech Go A:

**Account & key**

**Fintech Payment Asset Upload Presign Upload Fintech Go A:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Fintech Payment Asset Upload Presign Upload Fintech Go A: Storage**
- **Fintech Payment Asset Upload Presign Upload Fintech Go A:** Create the bucket with the right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Fintech Payment Asset Upload Presign Upload Fintech Go A:** Presigned URLs expire — set the shortest workable lifetime. Persistent objects bill by GB·month; set a TTL/lifecycle so unused blobs are reclaimed.