package cmd

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name    string
		current string
		latest  string
		want    int
		wantOK  bool
	}{
		{"older", "1.1.0", "v1.1.1", -1, true},
		{"equal", "v1.1.0", "1.1.0", 0, true},
		{"newer", "1.1.0", "1.0.9", 1, true},
		{"prerelease", "1.1.0", "v1.1.1-beta.1", -1, true},
		{"dev", "dev", "v1.0.4", 0, false},
		{"devel", "(devel)", "v1.0.4", 0, false},
		{"bad", "1.0", "v1.0.4", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := compareVersions(tt.current, tt.latest)
			if ok != tt.wantOK || got != tt.want {
				t.Fatalf("compareVersions(%q, %q) = (%d, %v), want (%d, %v)", tt.current, tt.latest, got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestUpdateCommands(t *testing.T) {
	tests := []struct {
		method string
		want   string
	}{
		{"npm", "npm install -g @ananke/cnstock-cli@latest"},
		{"go", "go install github.com/fatecannotbealtered/cnstock-cli/cmd/cnstock-cli@latest"},
		{"github", "Download the latest binary from https://github.com/fatecannotbealtered/cnstock-cli/releases/latest"},
		{"unknown", "npm install -g @ananke/cnstock-cli@latest"},
	}
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			got := updateCommands(tt.method)
			if len(got) == 0 || got[0] != tt.want {
				t.Fatalf("updateCommands(%q)[0] = %q, want %q", tt.method, got, tt.want)
			}
		})
	}
}

func TestSamePath(t *testing.T) {
	if !samePath(".", ".") {
		t.Fatal("samePath should match identical paths")
	}
	if samePath(".", "") {
		t.Fatal("samePath should reject empty paths")
	}
}
