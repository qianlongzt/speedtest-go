package web

import (
	"regexp"
	"testing"
)

func TestTrimPrefix(t *testing.T) {
	tests := []struct {
		name string
		url  string
		base string
		want string
	}{
		{"empty URL and base", "", "", ""},
		{"URL without base prefix", "/path/to/resource", "/base", "/path/to/resource"},
		{"URL with base prefix", "/base/path/to/resource", "/base", "path/to/resource"},
		{"URL with multiple base prefixes", "/base/base/path/to/resource", "/base", "base/path/to/resource"},
		{"base is a substring of URL", "/base/path/to/base/resource", "/base", "path/to/base/resource"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimPrefix(tt.url, tt.base)
			if got != tt.want {
				t.Errorf("trimPrefix(%q, %q) = %q, want %q", tt.url, tt.base, got, tt.want)
			}
		})
	}
}

func BenchmarkTrimPrefix(b *testing.B) {
	for i := 0; i < b.N; i++ {
		trimPrefix("/foo/bar", "/foo")
	}
}
func BenchmarkTrimPrefixReg(b *testing.B) {
	var removeBaseURL = regexp.MustCompile("^" + "/foo" + "/")
	for i := 0; i < b.N; i++ {
		removeBaseURL.ReplaceAllString("/foo/bar", "/foo")
	}
}
