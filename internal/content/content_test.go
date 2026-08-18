package content

import (
	"os"
	"path/filepath"
	"strings"
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

// TestParseWork exercises work parsing against its own testdata fixture, for
// the same reason the two above do: the fetched pack is gitignored and only
// present after `make content`.
func TestParseWork(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "work.yaml"))
	if err != nil {
		t.Fatalf("read testdata fixture: %v", err)
	}

	work, err := parseWork(b)
	if err != nil {
		t.Fatalf("parseWork: %v", err)
	}
	if got, want := len(work), 2; got != want {
		t.Fatalf("parsed %d roles, want %d", got, want)
	}

	first := work[0]
	if got, want := first.Company, "Acme Corp"; got != want {
		t.Errorf("Company = %q, want %q", got, want)
	}
	if got, want := first.Role, "Test Engineer"; got != want {
		t.Errorf("Role = %q, want %q", got, want)
	}
	if got, want := first.Location, "Testville, TS / Remote"; got != want {
		t.Errorf("Location = %q, want %q", got, want)
	}
	// A role still held writes no end date at all, so the absence is the fact
	// and the surface is what has a word for it.
	if got, want := first.Start, "2024-01"; got != want {
		t.Errorf("Start = %q, want %q", got, want)
	}
	if first.End != "" {
		t.Errorf("End = %q for an open-ended role, want it empty", first.End)
	}
	if got, want := len(first.Highlights), 2; got != want {
		t.Errorf("parsed %d highlights, want %d", got, want)
	}
	if got, want := work[1].End, "2023-12"; got != want {
		t.Errorf("End = %q, want %q", got, want)
	}
	if got, want := work[1].Project, "base-thing"; got != want {
		t.Errorf("Project = %q, want %q", got, want)
	}
}

// TestNamedLinksReadBothShapes is the finding a prototype round cost: a name in
// an entry's links block holds either one link or a list of them. Both are read
// into the same flat, ordered list, keyed by the name, and a link with no label
// of its own is shown under that name rather than under nothing.
func TestNamedLinksReadBothShapes(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "work.yaml"))
	if err != nil {
		t.Fatalf("read testdata fixture: %v", err)
	}
	work, err := parseWork(b)
	if err != nil {
		t.Fatalf("parseWork: %v", err)
	}

	want := []Link{
		{Slug: "code", Label: "first repo", URL: "https://example.com/acme/one"},
		{Slug: "code", Label: "code", URL: "https://example.com/acme/two"},
		{Slug: "write_up", Label: "Write-up", URL: "https://example.com/acme/writeup"},
	}
	got := work[0].Links
	if len(got) != len(want) {
		t.Fatalf("parsed %d links, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("link %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	if work[1].Links != nil {
		t.Errorf("a role with no links block parsed %+v", work[1].Links)
	}
}

// TestNamedLinksRejectAnythingButAMapping: the block is keyed by permanent
// names, and a pack that wrote it as a bare list has lost them, so it fails
// loudly at load rather than rendering a section with no names in it.
func TestNamedLinksRejectAnythingButAMapping(t *testing.T) {
	_, err := parseWork([]byte("work:\n  - slug: x\n    links:\n      - url: https://example.com\n"))
	if err == nil {
		t.Fatal("a links block written as a list parsed without complaint")
	}
	if !strings.Contains(err.Error(), "mapping") {
		t.Errorf("the error is %q, want it to say what shape was wanted", err)
	}
}
