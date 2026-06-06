package tui

import "testing"

func TestMaskSecretShort(t *testing.T) {
	if got := maskSecret("abcd"); got != "****" {
		t.Fatalf("maskSecret short = %q, want ****", got)
	}
}

func TestMaskSecretLong(t *testing.T) {
	if got := maskSecret("abcdef"); got != "ab**ef" {
		t.Fatalf("maskSecret long = %q, want ab**ef", got)
	}
}
