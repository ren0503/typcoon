package auth

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestBuildAuthAuditEntryUsesAllowlistedDetails(t *testing.T) {
	accountID := uuid.New()
	entry, err := buildAuthAuditEntry(authAuditEntry{
		Action:    ActionSessionIssued,
		ActorType: ActorAccount,
		ActorID:   accountID.String(),
		AccountID: accountID,
		Entity:    "account",
		Details:   authAuditDetails{Client: ClientGuestMobile, Provider: "firebase"},
	})
	if err != nil {
		t.Fatalf("buildAuthAuditEntry: %v", err)
	}
	if entry.Source != "cloud" || entry.Action != ActionSessionIssued {
		t.Fatalf("entry = %+v", entry)
	}
	if entry.ActorID == nil || *entry.ActorID != accountID.String() ||
		entry.Entity == nil || *entry.Entity != "account" ||
		entry.EntityID == nil || *entry.EntityID != accountID.String() {
		t.Fatalf("actor/entity = %+v", entry)
	}
	var details map[string]string
	if err := json.Unmarshal(entry.Details, &details); err != nil {
		t.Fatalf("unmarshal details: %v", err)
	}
	if details["client"] != ClientGuestMobile || details["provider"] != "firebase" {
		t.Fatalf("details = %+v", details)
	}
	if strings.Contains(string(entry.Details), "token") || strings.Contains(string(entry.Details), "hash") {
		t.Fatalf("details contain secret-like fields: %s", entry.Details)
	}
}
