package application

import (
	"strings"
	"testing"
)

func TestNormalizeGitRepository(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"full https URL", "https://github.com/org/repo", "https://github.com/org/repo"},
		{"full http URL", "http://github.com/org/repo", "http://github.com/org/repo"},
		{"SSH URL", "git@github.com:org/repo.git", "git@github.com:org/repo.git"},
		{"domain-prefixed", "github.com/org/repo", "github.com/org/repo"},
		{"gitlab domain", "gitlab.com/org/repo", "gitlab.com/org/repo"},
		{"bare slug", "org/repo", "https://github.com/org/repo"},
		{"bare slug with .git", "org/repo.git", "https://github.com/org/repo.git"},
		{"bare slug nested", "org/repo/subdir", "https://github.com/org/repo/subdir"},
		{"empty string", "", ""},
		{"single word", "myrepo", "myrepo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := normalizeGitRepository(tt.in)
			if got != tt.want {
				t.Errorf("normalizeGitRepository(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func FuzzNormalizeGitRepository(f *testing.F) {
	// Seed with representative inputs from each branch.
	for _, s := range []string{
		"", "myrepo", "org/repo", "org/repo.git",
		"https://github.com/org/repo", "http://gitlab.com/a/b",
		"git@github.com:org/repo.git", "github.com/org/repo",
		"gitlab.com/org/repo", "org/repo/sub",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		result := normalizeGitRepository(input)

		// Invariant 1: must never panic (implicit).

		// Invariant 2: idempotent.
		if second := normalizeGitRepository(result); second != result {
			t.Errorf("not idempotent: f(%q)=%q, f(f(%q))=%q", input, result, input, second)
		}

		// Invariant 3: if the input already had a scheme, it is preserved.
		if strings.Contains(input, "://") && result != input {
			t.Errorf("scheme-bearing input mutated: %q -> %q", input, result)
		}

		// Invariant 4: if the input starts with git@, it is preserved.
		if strings.HasPrefix(input, "git@") && result != input {
			t.Errorf("SSH input mutated: %q -> %q", input, result)
		}
	})
}
