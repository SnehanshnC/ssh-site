package content

import (
	"os"
	"path/filepath"
	"testing"
)

// TestParseIdentity exercises identity parsing against a small hand-written
// fixture in testdata/, so it never depends on the fetched content pack
// files (which are gitignored and only present after `make content`).
func TestParseIdentity(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "identity.yaml"))
	if err != nil {
		t.Fatalf("read testdata fixture: %v", err)
	}

	identity, err := parseIdentity(b)
	if err != nil {
		t.Fatalf("parseIdentity: %v", err)
	}

	if got, want := identity.Name, "Test Person"; got != want {
		t.Errorf("Name = %q, want %q", got, want)
	}
	if got, want := identity.Role.Title, "Test Engineer"; got != want {
		t.Errorf("Role.Title = %q, want %q", got, want)
	}
	if got, want := identity.Role.Company, "Test Co"; got != want {
		t.Errorf("Role.Company = %q, want %q", got, want)
	}
	if got, want := identity.Role.Program, "Test Program S00"; got != want {
		t.Errorf("Role.Program = %q, want %q", got, want)
	}
	if got, want := identity.Education.Institution, "Test University"; got != want {
		t.Errorf("Education.Institution = %q, want %q", got, want)
	}
	if got, want := identity.Education.StartYear, 2020; got != want {
		t.Errorf("Education.StartYear = %d, want %d", got, want)
	}
	if got, want := identity.Education.EndYear, 2024; got != want {
		t.Errorf("Education.EndYear = %d, want %d", got, want)
	}
	if want := []string{"First tagline.", "Second tagline."}; !equalStrings(identity.Taglines, want) {
		t.Errorf("Taglines = %v, want %v", identity.Taglines, want)
	}
	if want := []string{"Go"}; !equalStrings(identity.Skills.Languages, want) {
		t.Errorf("Skills.Languages = %v, want %v", identity.Skills.Languages, want)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestParseLinks exercises links parsing against its own testdata fixture, for
// the same reason TestParseIdentity does: the fetched pack is gitignored and
// only present after `make content`.
func TestParseLinks(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "links.yaml"))
	if err != nil {
		t.Fatalf("read testdata fixture: %v", err)
	}

	links, err := parseLinks(b)
	if err != nil {
		t.Fatalf("parseLinks: %v", err)
	}
	if got, want := len(links), 2; got != want {
		t.Fatalf("parsed %d links, want %d", got, want)
	}

	pack := &Pack{Links: links}
	link, ok := pack.Link("linkedin")
	if !ok {
		t.Fatal("Link(\"linkedin\") not found")
	}
	if got, want := link.URL, "https://www.linkedin.com/in/test-person-0a1b2c3d4/"; got != want {
		t.Errorf("URL = %q, want %q", got, want)
	}
	if got, want := link.Label, "LinkedIn"; got != want {
		t.Errorf("Label = %q, want %q", got, want)
	}
	if _, ok := pack.Link("nope"); ok {
		t.Error("Link(\"nope\") found something")
	}
}
