package beadmail

import (
	"testing"

	"github.com/gastownhall/gascity/internal/beads"
	"github.com/gastownhall/gascity/internal/mail"
	"github.com/gastownhall/gascity/internal/session"
)

// TestReplyGoesToDeclaredReplyAddressAfterPoolSenderCloses is the pool
// session case: the sender has no alias, mails human, declares a durable reply
// address, and closes. The human's reply must land in a live mailbox, not in
// the closed session's.
func TestReplyGoesToDeclaredReplyAddressAfterPoolSenderCloses(t *testing.T) {
	store := beads.NewMemStore()
	p := newTestProvider(store)

	pool, err := store.Create(beads.Bead{
		Type:   session.BeadType,
		Labels: []string{session.LabelSession},
		Metadata: map[string]string{
			"session_name": "gastown__polecat-opus-ga-pool",
			"template":     "lspa-api/gastown.polecat-opus",
		},
	})
	if err != nil {
		t.Fatalf("Create pool session: %v", err)
	}
	if _, err := store.Create(beads.Bead{
		Type:     session.BeadType,
		Labels:   []string{session.LabelSession},
		Metadata: map[string]string{"alias": "gastown.mayor", "session_name": "gastown__mayor"},
	}); err != nil {
		t.Fatalf("Create mayor session: %v", err)
	}

	sent, err := p.SendWithReplyAddress(pool.ID, "human", "Grill round 1", "questions", "gastown.mayor")
	if err != nil {
		t.Fatalf("SendWithReplyAddress: %v", err)
	}
	if sent.From != "gastown__polecat-opus-ga-pool" {
		t.Fatalf("sent From = %q, want the pool session's own name", sent.From)
	}
	if err := store.Close(pool.ID); err != nil {
		t.Fatalf("Close pool session: %v", err)
	}

	reply, err := p.Reply(sent.ID, "human", "", "answers")
	if err != nil {
		t.Fatalf("Reply: %v", err)
	}
	if reply.To != "gastown.mayor" {
		t.Fatalf("reply To = %q, want the declared reply address", reply.To)
	}
	b, err := store.Get(reply.ID)
	if err != nil {
		t.Fatalf("Get reply: %v", err)
	}
	if b.Assignee != "gastown.mayor" {
		t.Fatalf("reply Assignee = %q, want gastown.mayor", b.Assignee)
	}
	if got := b.Metadata[toSessionIDMetadataKey]; got != "" {
		t.Fatalf("reply %s = %q, want none: the closed sender is not the recipient", toSessionIDMetadataKey, got)
	}
	msgs, err := p.Inbox("gastown.mayor")
	if err != nil {
		t.Fatalf("Inbox: %v", err)
	}
	if len(msgs) != 1 || msgs[0].ID != reply.ID {
		t.Fatalf("mayor inbox = %+v, want only reply %s", msgs, reply.ID)
	}
}

// TestReplyAddressWithoutSenderSession covers a sender that resolves to no
// session (no from_session_id): the declared address still wins.
func TestReplyAddressWithoutSenderSession(t *testing.T) {
	store := beads.NewMemStore()
	p := newTestProvider(store)

	sent, err := p.SendWithReplyAddress("alice", "bob", "Hello", "body", "carol")
	if err != nil {
		t.Fatalf("SendWithReplyAddress: %v", err)
	}
	b, err := store.Get(sent.ID)
	if err != nil {
		t.Fatalf("Get sent: %v", err)
	}
	if got := b.Metadata[mail.ReplyAddressMetadataKey]; got != "carol" {
		t.Fatalf("sent %s = %q, want carol", mail.ReplyAddressMetadataKey, got)
	}
	if got := b.Metadata[fromSessionIDMetadataKey]; got != "" {
		t.Fatalf("sent %s = %q, want none", fromSessionIDMetadataKey, got)
	}

	reply, err := p.Reply(sent.ID, "bob", "", "hi")
	if err != nil {
		t.Fatalf("Reply: %v", err)
	}
	if reply.To != "carol" {
		t.Fatalf("reply To = %q, want carol", reply.To)
	}
}

func TestSendWithReplyAddressRequiresAddress(t *testing.T) {
	p := newTestProvider(beads.NewMemStore())
	if _, err := p.SendWithReplyAddress("alice", "bob", "s", "b", " "); err == nil {
		t.Fatal("SendWithReplyAddress with blank address succeeded, want error")
	}
}
