package health

import (
	"errors"
	"testing"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want FailureKind
	}{
		{name: "auth", err: errors.New("session expired"), want: FailureAuth},
		{name: "network", err: errors.New("proxy connection reset"), want: FailureNetwork},
		{name: "unknown is not auth", err: errors.New("unexpected response"), want: FailureUnknown},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Classify(test.err); got != test.want {
				t.Fatalf("Classify() = %q, want %q", got, test.want)
			}
		})
	}
}
