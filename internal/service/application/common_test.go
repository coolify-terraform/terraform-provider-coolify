package application

import "testing"

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
