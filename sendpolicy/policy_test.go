package sendpolicy

import (
	"errors"
	"testing"
)

func TestPolicyCountsMessages(t *testing.T) {
	p := New(2)
	if err := p.Allow("a1"); err != nil {
		t.Fatal(err)
	}
	if err := p.Allow("a1"); err != nil {
		t.Fatal(err)
	}
	if err := p.Allow("a1"); !errors.Is(err, ErrDailyLimit) {
		t.Fatalf("got %v, want ErrDailyLimit", err)
	}
}
