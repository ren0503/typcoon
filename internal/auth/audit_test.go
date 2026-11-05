package auth

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/ren0503/tycoon/internal/audit"
)

type captureAudit struct {
	entries []audit.Entry
}

func (c *captureAudit) AppendTx(_ context.Context, _ audit.Execer, entry audit.Entry) error {
	c.entries = append(c.entries, entry)
	return nil
}

func TestAppendAuthAuditUsesAllowlistedDetails(t *testing.T) {
	recorder := &captureAudit{}
	accountID := uuid.New()
	err := appendAuthAudit(context.Background(), recorder, nil, authAuditEntry{
		Action:    ActionSessionIssued,
		ActorType: ActorAccount,
		ActorID:   accountID.String(),
		AccountID: accountID,
		Entity:    "account",
		Details:   authAuditDetails{Client: ClientGuestMobile, Provider: "firebase"},
	})
	if err != nil {
		t.Fatalf("appendAuthAudit: %v", err)
	}
	if len(recorder.entries) != 1 {
		t.Fatalf("entries = %d", len(recorder.entries))
	}
	entry := recorder.entries[0]
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
