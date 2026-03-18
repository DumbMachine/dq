package config

import (
	"testing"

	"github.com/zalando/go-keyring"
)

func init() {
	// Use in-memory keyring for all tests
	keyring.MockInit()
}

func TestResolvePassword_Empty(t *testing.T) {
	pw, isPlain, err := ResolvePassword("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pw != "" || isPlain {
		t.Fatalf("expected empty/non-plain, got %q/%v", pw, isPlain)
	}
}

func TestResolvePassword_EnvVar(t *testing.T) {
	t.Setenv("DQ_TEST_PW", "secret123")
	pw, isPlain, err := ResolvePassword("env:DQ_TEST_PW")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pw != "secret123" || isPlain {
		t.Fatalf("expected secret123/false, got %q/%v", pw, isPlain)
	}
}

func TestResolvePassword_EnvVar_NotSet(t *testing.T) {
	_, _, err := ResolvePassword("env:DQ_TEST_NONEXISTENT_VAR")
	if err == nil {
		t.Fatal("expected error for unset env var")
	}
}

func TestResolvePassword_Plaintext(t *testing.T) {
	pw, isPlain, err := ResolvePassword("mypassword")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pw != "mypassword" || !isPlain {
		t.Fatalf("expected mypassword/true, got %q/%v", pw, isPlain)
	}
}

func TestResolvePassword_Keyring(t *testing.T) {
	if err := StoreInKeyring("testconn", "keyring-secret"); err != nil {
		t.Fatalf("StoreInKeyring: %v", err)
	}

	pw, isPlain, err := ResolvePassword("keyring:testconn")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pw != "keyring-secret" || isPlain {
		t.Fatalf("expected keyring-secret/false, got %q/%v", pw, isPlain)
	}
}

func TestResolvePassword_Keyring_NotFound(t *testing.T) {
	_, _, err := ResolvePassword("keyring:nonexistent")
	if err == nil {
		t.Fatal("expected error for missing keyring entry")
	}
}

func TestStoreAndDeleteKeyring(t *testing.T) {
	if err := StoreInKeyring("deltest", "pw"); err != nil {
		t.Fatalf("StoreInKeyring: %v", err)
	}

	// Verify it's there
	pw, _, err := ResolvePassword("keyring:deltest")
	if err != nil || pw != "pw" {
		t.Fatalf("expected pw, got %q (err=%v)", pw, err)
	}

	// Delete it
	if err := DeleteFromKeyring("deltest"); err != nil {
		t.Fatalf("DeleteFromKeyring: %v", err)
	}

	// Verify it's gone
	_, _, err = ResolvePassword("keyring:deltest")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestDeleteFromKeyring_NotFound(t *testing.T) {
	// Deleting a non-existent entry should not error
	if err := DeleteFromKeyring("nope"); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}
