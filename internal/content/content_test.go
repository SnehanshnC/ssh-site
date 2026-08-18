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

// TestParseProjects exercises the projects section against its own testdata
// fixture, for the same reason the three above do: the fetched pack is
// gitignored and only present after `make content`.
//
// The section's one file carries three lists, so what is checked is that all
// three come out of one parse - the awards list flat and keyed to a project by
// slug rather than nested under it, and the programs list read without
// complaint even though nothing renders it.
func TestParseProjects(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("testdata", "projects.yaml"))
	if err != nil {
		t.Fatalf("read testdata fixture: %v", err)
	}

	doc, err := parseProjects(b)
	if err != nil {
		t.Fatalf("parseProjects: %v", err)
	}
	if got, want := len(doc.Projects), 2; got != want {
		t.Fatalf("parsed %d projects, want %d", got, want)
	}
	if got, want := len(doc.Awards), 5; got != want {
		t.Fatalf("parsed %d awards, want %d", got, want)
	}
	if got, want := len(doc.Programs), 2; got != want {
		t.Fatalf("parsed %d programs, want %d", got, want)
	}

	first := doc.Projects[0]
	if got, want := first.Name, "Test Thing"; got != want {
		t.Errorf("Name = %q, want %q", got, want)
	}
	if got, want := first.Summary, "A thing built to be tested."; got != want {
		t.Errorf("Summary = %q, want %q", got, want)
	}
	if got, want := len(first.Highlights), 2; got != want {
		t.Errorf("parsed %d highlights, want %d", got, want)
	}
	if want := []string{"Go", "YAML"}; !equalStrings(first.Stack, want) {
		t.Errorf("Stack = %v, want %v", first.Stack, want)
	}
	if want := []string{"Test Teammate"}; !equalStrings(first.Teammates, want) {
		t.Errorf("Teammates = %v, want %v", first.Teammates, want)
	}
	if got, want := first.BuiltAt, "TestHacks"; got != want {
		t.Errorf("BuiltAt = %q, want %q", got, want)
	}
	if got, want := first.Date, "2025-01"; got != want {
		t.Errorf("Date = %q, want %q", got, want)
	}
	if want := []string{"Noted for the record."}; !equalStrings(first.Notes, want) {
		t.Errorf("Notes = %v, want %v", first.Notes, want)
	}

	// The entry links block is content.NamedLinks, the same reader work.yaml
	// goes through, so a name holding a list of repos reads here too.
	wantLinks := []Link{
		{Slug: "code", Label: "first repo", URL: "https://example.com/test-thing/one"},
		{Slug: "code", Label: "code", URL: "https://example.com/test-thing/two"},
		{Slug: "write_up", Label: "Write-up", URL: "https://example.com/test-thing/writeup"},
	}
	if len(first.Links) != len(wantLinks) {
		t.Fatalf("parsed %d links, want %d: %+v", len(first.Links), len(wantLinks), first.Links)
	}
	for i := range wantLinks {
		if first.Links[i] != wantLinks[i] {
			t.Errorf("link %d = %+v, want %+v", i, first.Links[i], wantLinks[i])
		}
	}

	// An award places or it is selected out of a field; the pack writes one or
	// the other, and the size of that field is prose as often as it is a number.
	award := doc.Awards[0]
	if got, want := award.Placement, "1st Place Overall"; got != want {
		t.Errorf("Placement = %q, want %q", got, want)
	}
	if got, want := award.Year, 2025; got != want {
		t.Errorf("Year = %d, want %d", got, want)
	}
	if got, want := award.Track, "Test Track"; got != want {
		t.Errorf("Track = %q, want %q", got, want)
	}
	if got, want := award.Participants, "40+ teams"; got != want {
		t.Errorf("Participants = %q, want %q", got, want)
	}
	if got, want := award.Prize, "$100"; got != want {
		t.Errorf("Prize = %q, want %q", got, want)
	}
	if want := []string{"Best Use of Testing"}; !equalStrings(award.Extras, want) {
		t.Errorf("Extras = %v, want %v", award.Extras, want)
	}
	if got, want := doc.Awards[1].Selection, "Test Select, 4 of 400"; got != want {
		t.Errorf("Selection = %q, want %q", got, want)
	}
	if doc.Awards[1].Placement != "" {
		t.Errorf("Placement = %q for a selection, want it empty", doc.Awards[1].Placement)
	}

	if got, want := doc.Programs[0].Name, "Test Program"; got != want {
		t.Errorf("Programs[0].Name = %q, want %q", got, want)
	}
	if got, want := doc.Programs[0].Detail, "Invited to test."; got != want {
		t.Errorf("Programs[0].Detail = %q, want %q", got, want)
	}
	if got, want := doc.Programs[1].Year, 2024; got != want {
		t.Errorf("Programs[1].Year = %d, want %d", got, want)
	}
}

// TestProjectResolvesBySlug is the awards relation followed forward: an award
// names the project it was won for, and the awards section opens that project's
// own page. A slug no project carries resolves to nothing at all, which is what
// lets a surface step around a reference the pack has yet to fix.
func TestProjectResolvesBySlug(t *testing.T) {
	pack := fixturePack(t)

	project, ok := pack.Project("test-thing")
	if !ok {
		t.Fatal("the first project did not resolve by its own slug")
	}
	if got, want := project.Name, "Test Thing"; got != want {
		t.Errorf("resolved %q, want %q", got, want)
	}
	if _, ok := pack.Project("no-such-thing"); ok {
		t.Error("a slug no project carries resolved to a project")
	}
	if _, ok := pack.Project(""); ok {
		t.Error("the empty slug resolved to a project")
	}

	// The two ends of the same relation: the fixture's orphan award names a
	// project the list does not carry, and its unattached one names none.
	for _, award := range pack.Awards {
		_, resolved := pack.Project(award.Project)
		if want := award.Slug != "orphan-award" && award.Slug != "unattached-award"; resolved != want {
			t.Errorf("%s resolved to %v, want %v", award.Slug, resolved, want)
		}
	}
}

// TestAwardsForResolvesBySlug. The awards list is flat and names the project it
// belongs to, so the relation is a lookup rather than a nesting - and it is
// written once here because two sections read it from opposite ends.
func TestAwardsForResolvesBySlug(t *testing.T) {
	pack := fixturePack(t)

	got := pack.AwardsFor("test-thing")
	if len(got) != 2 {
		t.Fatalf("resolved %d awards, want 2: %+v", len(got), got)
	}
	// In the order the pack wrote them, which is the order they are shown in.
	if got[0].Slug != "test-thing-first" || got[1].Slug != "test-thing-selected" {
		t.Errorf("resolved %q then %q, want them in pack order", got[0].Slug, got[1].Slug)
	}
	for _, award := range got {
		if award.Project != "test-thing" {
			t.Errorf("%s belongs to %q", award.Slug, award.Project)
		}
	}

	if got := pack.AwardsFor("bare-thing"); len(got) != 1 {
		t.Errorf("resolved %d awards for the second project, want 1", len(got))
	}
	// An award naming a project the list does not carry belongs to no page, and
	// an empty slug is not a wildcard onto the ones that name nothing.
	if got := pack.AwardsFor("no-project-of-that-name"); got != nil {
		t.Errorf("resolved %+v for a slug no project carries", got)
	}
	if got := pack.AwardsFor(""); got != nil {
		t.Errorf("the empty slug resolved %+v", got)
	}
}

// fixturePack is the projects fixture as a Pack, for the two lookups that read
// its lists rather than its parsing. It is the fixture rather than the fetched
// pack for the same reason every parser test here is: the fetched pack is
// gitignored and only present after `make content`.
func fixturePack(t *testing.T) *Pack {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "projects.yaml"))
	if err != nil {
		t.Fatalf("read testdata fixture: %v", err)
	}
	doc, err := parseProjects(b)
	if err != nil {
		t.Fatalf("parseProjects: %v", err)
	}
	return &Pack{Projects: doc.Projects, Awards: doc.Awards, Programs: doc.Programs}
}
