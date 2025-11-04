package printable

import (
	"testing"

	"github.com/coscene-io/cocli/internal/printer/table"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestRegistryCredentialToProtoMessage(t *testing.T) {
	cred := NewRegistryCredential("user", "pass")
	msg := cred.ToProtoMessage()

	st, ok := msg.(*structpb.Struct)
	if !ok {
		t.Fatalf("expected *structpb.Struct, got %T", msg)
	}

	username := st.Fields["username"].GetStringValue()
	password := st.Fields["password"].GetStringValue()

	if username != "user" {
		t.Fatalf("expected username 'user', got %q", username)
	}
	if password != "pass" {
		t.Fatalf("expected password 'pass', got %q", password)
	}
}

func TestRegistryCredentialToTable(t *testing.T) {
	cred := NewRegistryCredential("alice", "secret")
	tbl := cred.ToTable(&table.PrintOpts{})

	if len(tbl.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(tbl.Rows))
	}

	if tbl.Rows[0][0] != "USERNAME" || tbl.Rows[0][1] != "alice" {
		t.Fatalf("unexpected username row: %#v", tbl.Rows[0])
	}
	if tbl.Rows[1][0] != "PASSWORD" || tbl.Rows[1][1] != "secret" {
		t.Fatalf("unexpected password row: %#v", tbl.Rows[1])
	}
}
