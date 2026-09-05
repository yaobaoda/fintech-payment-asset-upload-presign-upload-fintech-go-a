package uploadpolicy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const MaxAssetBytes int64 = 10 * 1024 * 1024

type Request struct {
	PaymentEventID string `json:"payment_event_id"`
	EventType      string `json:"event_type"`
	AssetKind      string `json:"asset_kind"`
	ContentType    string `json:"content_type"`
	SizeBytes      int64  `json:"size_bytes"`
	RiskLevel      string `json:"risk_level"`
}

type Decision struct {
	ID        string `json:"decision_id"`
	Action    string `json:"action"`
	Reason    string `json:"reason"`
	ObjectKey string `json:"object_key,omitempty"`
}

func Authorize(r Request) Decision {
	id := stableID(r)
	decision := Decision{ID: id, Action: "reject", Reason: "invalid_request"}

	if strings.TrimSpace(r.PaymentEventID) == "" || strings.TrimSpace(r.ContentType) == "" || r.SizeBytes <= 0 {
		return decision
	}
	if r.EventType != "payment_settled" {
		decision.Reason = "payment_not_settled"
		return decision
	}
	if r.RiskLevel == "high" {
		decision.Action = "manual_review"
		decision.Reason = "risk_review_required"
		return decision
	}
	if r.RiskLevel != "low" && r.RiskLevel != "medium" {
		decision.Reason = "invalid_risk_level"
		return decision
	}
	if r.AssetKind != "receipt" && r.AssetKind != "supporting_document" {
		decision.Reason = "asset_kind_not_allowed"
		return decision
	}
	if r.SizeBytes > MaxAssetBytes {
		decision.Reason = "asset_too_large"
		return decision
	}

	decision.Action = "issue_upload_url"
	decision.Reason = "policy_approved"
	decision.ObjectKey = fmt.Sprintf("payment-events/%s/%s-%s", safePart(r.PaymentEventID), r.AssetKind, id)
	return decision
}

func stableID(r Request) string {
	value := strings.Join([]string{
		r.PaymentEventID,
		r.EventType,
		r.AssetKind,
		r.ContentType,
		fmt.Sprintf("%d", r.SizeBytes),
		r.RiskLevel,
	}, "\x00")
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:8])
}

func safePart(value string) string {
	var b strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}
