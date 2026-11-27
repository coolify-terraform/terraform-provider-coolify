package client

import (
	"strings"
	"testing"
)

func TestReadLimitedBodyN(t *testing.T) {
	t.Parallel()
	got, err := readLimitedBodyN(strings.NewReader("abcd"), 4)
	if err != nil {
		t.Fatalf("exact limit: %v", err)
	}
	if string(got) != "abcd" {
		t.Fatalf("exact limit body = %q", got)
	}
	if _, err := readLimitedBodyN(strings.NewReader("abcde"), 4); err == nil || !strings.Contains(err.Error(), "exceeds 4 bytes") {
		t.Fatalf("oversize err = %v", err)
	}
}
