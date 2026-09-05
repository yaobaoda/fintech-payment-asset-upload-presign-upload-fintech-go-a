package uploadpolicy

import "testing"

func TestAuthorizePaymentAsset(t *testing.T) {
	tests := []struct {
		name      string
		request   Request
		action    string
		reason    string
		hasObject bool
	}{
		{
			name:    "settled payment receipt",
			request: Request{PaymentEventID: "evt_1042", EventType: "payment_settled", AssetKind: "receipt", ContentType: "application/pdf", SizeBytes: 2048, RiskLevel: "low"},
			action:  "issue_upload_url", reason: "policy_approved", hasObject: true,
		},
		{
			name:    "high risk event requires review",
			request: Request{PaymentEventID: "evt_1043", EventType: "payment_settled", AssetKind: "receipt", ContentType: "application/pdf", SizeBytes: 2048, RiskLevel: "high"},
			action:  "manual_review", reason: "risk_review_required",
		},
		{
			name:    "unsettled payment is rejected",
			request: Request{PaymentEventID: "evt_1044", EventType: "payment_failed", AssetKind: "receipt", ContentType: "application/pdf", SizeBytes: 2048, RiskLevel: "low"},
			action:  "reject", reason: "payment_not_settled",
		},
		{
			name:    "oversized evidence is rejected",
			request: Request{PaymentEventID: "evt_1045", EventType: "payment_settled", AssetKind: "supporting_document", ContentType: "image/png", SizeBytes: MaxAssetBytes + 1, RiskLevel: "medium"},
			action:  "reject", reason: "asset_too_large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Authorize(tt.request)
			if got.Action != tt.action || got.Reason != tt.reason {
				t.Fatalf("Authorize() = action %q, reason %q; want %q, %q", got.Action, got.Reason, tt.action, tt.reason)
			}
			if (got.ObjectKey != "") != tt.hasObject {
				t.Fatalf("Authorize() object key presence = %v; want %v", got.ObjectKey != "", tt.hasObject)
			}
			if got.ID != Authorize(tt.request).ID {
				t.Fatal("Authorize() decision ID changed for identical input")
			}
		})
	}
}
