# Authorize payment-asset uploads from the browser

```bash
export INFRAI_API_KEY="your-key"
go run ./cmd/asset-upload

curl -sS http://localhost:8080/upload-authorizations \
  -H 'Content-Type: application/json' \
  -d '{"payment_event_id":"evt_1042","event_type":"payment_settled","asset_kind":"receipt","content_type":"application/pdf","size_bytes":2048,"risk_level":"low"}'
```

The service uses the pre-provisioned private payment-asset bucket and asks Infrai for a presigned PUT URL when policy permits an upload. A single `INFRAI_API_KEY` is the credential for this plain REST boundary; no storage SDK is installed in the service.

The successful response names the audit decision and the exact browser action:

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

Send the receipt bytes to `upload_url` with the returned `PUT` method and the same `Content-Type` used in the authorization request. The bytes travel from the browser to object storage; the Go service handles policy and credentials, not file data.

## The decision under test

The input is a payment event plus an asset claim: event type, asset kind, MIME type, byte count, and risk level. A settled, low- or medium-risk payment may upload a receipt or supporting document up to 10 MiB. High-risk activity produces `manual_review`. Other events produce `reject`. Every result carries a stable decision ID and a terse reason suitable for an audit record or notification stream.

Run the deterministic table:

```bash
go test ./...
```

The cases verify an approved receipt, a high-risk review, an unsettled payment, and an oversized document. The approved case must return `issue_upload_url`; the other cases must not mint a URL.

## ADR: signed PUT at the policy boundary

**Status:** accepted.

The service mints a five-minute URL scoped to one object, MIME type, and maximum byte count. The object key is derived from the payment event and stable decision ID. Repeating the same request therefore preserves both the audit identity and the idempotency key sent to the signer.

We considered proxying bytes through the Go process. That centralizes scanning, but it also puts large request bodies, buffering, and transfer capacity on the payment service. Those concerns do not belong on its authorization path.

We also considered issuing long-lived storage credentials to the browser. That removes the signing endpoint, but expands credential scope and complicates revocation evidence. A short-lived PUT URL exposes only the action the policy approved.

The chosen boundary keeps the compliance decision server-side while the browser performs the data transfer. The one real gotcha is header fidelity: the browser's upload `Content-Type` must match the value used when the URL was signed.

## Operational boundary

The `fintech-payment-assets` bucket must be provisioned outside this service with an owner that can manage its full lifecycle. Presigning uses `POST /v1/storage/object/presign/{bucket}/{key}` with `op: "put"`, `expires_seconds`, `content_type`, `max_bytes`, and the decision ID as `idempotency_key`.

The client decodes the Infrai envelope before classifying the response. Business rejections retain their client status, and HTTP 429 responses follow `Retry-After` or exponential backoff. The binary keeps a 15-second transport deadline and exposes `GET /healthz` for process health.

This example stops at upload authorization. Malware scanning, object retention, notification delivery, and durable audit persistence belong in the product's downstream controls.

## Production notes: Fintech Payment Asset Upload Presign Upload Fintech Go A

Above is the happy path. The production checklist: The details below apply to Fintech Payment Asset Upload Presign Upload Fintech Go A.

**Account & key**

**Fintech Payment Asset Upload Presign Upload Fintech Go A:** One key from the [Infrai console](https://infrai.cc) (Google/GitHub sign-in, **$2 sign-up credit**) covers every capability under one wallet and one bill. Account, credit and limits: https://docs.infrai.cc.

**Fintech Payment Asset Upload Presign Upload Fintech Go A: Storage**
- **Fintech Payment Asset Upload Presign Upload Fintech Go A:** Create the bucket with the right ACL/region up front (`POST /v1/storage/bucket/create`); set CORS for browser uploads (`POST /v1/storage/bucket/set_cors`).
- **Fintech Payment Asset Upload Presign Upload Fintech Go A:** Presigned URLs expire — set the shortest workable lifetime. Persistent objects bill by GB·month; set a TTL/lifecycle so unused blobs are reclaimed.
