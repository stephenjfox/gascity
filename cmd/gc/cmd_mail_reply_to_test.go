package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gastownhall/gascity/internal/beads"
	"github.com/gastownhall/gascity/internal/mail"
	"github.com/gastownhall/gascity/internal/session"
)

// TestCmdMailSendReplyToStoresResolvedReplyAddress: --reply-to resolves like a
// recipient and is stored on the message, so a reply goes to the live mayor
// rather than to the sender.
func TestCmdMailSendReplyToStoresResolvedReplyAddress(t *testing.T) {
	t.Setenv("GC_BEADS", "file")
	t.Setenv("GC_MAIL", "")
	t.Setenv("GC_ALIAS", "")
	t.Setenv("GC_SESSION_ID", "")
	t.Setenv("GC_AGENT", "")

	cityPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(cityPath, "city.toml"), []byte("[workspace]\nname = \"test-city\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(city.toml): %v", err)
	}
	t.Setenv("GC_CITY", cityPath)

	store, err := openCityStoreAt(cityPath)
	if err != nil {
		t.Fatalf("openCityStoreAt: %v", err)
	}
	if _, err := store.Create(beads.Bead{
		Type:   session.BeadType,
		Labels: []string{session.LabelSession},
		Metadata: map[string]string{
			namedSessionIdentityMetadata: "test-city/mayor",
			"alias":                      "mayor",
			"session_name":               "mayor-session",
		},
	}); err != nil {
		t.Fatalf("Create mayor: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := cmdMailSendJSON([]string{"human"}, false, false, "", "", "Grill round 1", "questions", "", "mayor/", false, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("cmdMailSendJSON() = %d, want 0; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	storeAfter, err := openCityStoreAt(cityPath)
	if err != nil {
		t.Fatalf("openCityStoreAt after send: %v", err)
	}
	msgs, err := storeAfter.List(beads.ListQuery{Type: "message", Status: "open", TierMode: beads.TierBoth})
	if err != nil {
		t.Fatalf("List messages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("messages = %#v, want one", msgs)
	}
	if got := msgs[0].Metadata[mail.ReplyAddressMetadataKey]; got != "mayor" {
		t.Fatalf("%s = %q, want the resolved mailbox %q", mail.ReplyAddressMetadataKey, got, "mayor")
	}
}

func TestCmdMailSendReplyToRefusesUnknownAddress(t *testing.T) {
	t.Setenv("GC_BEADS", "file")
	t.Setenv("GC_MAIL", "")
	t.Setenv("GC_ALIAS", "")
	t.Setenv("GC_SESSION_ID", "")
	t.Setenv("GC_AGENT", "")
	cityPath := t.TempDir()
	if err := os.WriteFile(filepath.Join(cityPath, "city.toml"), []byte("[workspace]\nname = \"test-city\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(city.toml): %v", err)
	}
	t.Setenv("GC_CITY", cityPath)

	var stdout, stderr bytes.Buffer
	if code := cmdMailSendJSON([]string{"human"}, false, false, "", "", "s", "b", "", "nobody", false, &stdout, &stderr); code == 0 {
		t.Fatalf("cmdMailSendJSON() = 0, want failure for an unknown reply address; stdout=%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unknown reply address") {
		t.Fatalf("stderr = %q, want unknown reply address", stderr.String())
	}
}

// TestWithReplyAddressRefusesProviderWithoutCapability: a provider that cannot
// store the address fails the send instead of dropping the address, which
// would orphan the reply again.
func TestWithReplyAddressRefusesProviderWithoutCapability(t *testing.T) {
	if _, err := withReplyAddress(mail.NewFake(), "mayor"); err == nil {
		t.Fatal("withReplyAddress(fake) succeeded, want error")
	}
}
