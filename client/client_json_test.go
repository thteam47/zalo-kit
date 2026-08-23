package client

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestQRScanUsesPublicJSONContract(t *testing.T) {
	raw, err := json.Marshal(QRScan{Scanned: true, Profile: map[string]any{"display_name": "Test"}})
	if err != nil {
		t.Fatal(err)
	}
	got := string(raw)
	if !strings.Contains(got, `"scanned":true`) || strings.Contains(got, `"Scanned"`) {
		t.Fatalf("unexpected QR scan JSON: %s", got)
	}
}
