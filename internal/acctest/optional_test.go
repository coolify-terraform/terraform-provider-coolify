package acctest

import (
	"strings"
	"testing"
)

func TestDecodeOptional_MissingKeyIsNil(t *testing.T) {
	t.Parallel()
	body, err := DecodeObject(strings.NewReader(`{"frequency":"daily"}`))
	if err != nil {
		t.Fatal(err)
	}
	var absent int
	got, err := DecodeOptional[int64](body, "timeout", &absent)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("timeout = %v, want nil", *got)
	}
	if absent != 1 {
		t.Fatalf("absent = %d, want 1", absent)
	}
}

func TestDecodeOptional_PresentValue(t *testing.T) {
	t.Parallel()
	body, err := DecodeObject(strings.NewReader(`{"timeout":120}`))
	if err != nil {
		t.Fatal(err)
	}
	var absent int
	got, err := DecodeOptional[int64](body, "timeout", &absent)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || *got != 120 {
		t.Fatalf("timeout = %v, want 120", got)
	}
	if absent != 0 {
		t.Fatalf("absent = %d, want 0", absent)
	}
}

func TestDecodeOptional_NullIsAbsent(t *testing.T) {
	t.Parallel()
	body, err := DecodeObject(strings.NewReader(`{"timeout":null}`))
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeOptional[int64](body, "timeout", nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatal("null timeout should be absent")
	}
}
