package content

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// The pack's projects section, whose one file carries three lists: the projects
// themselves, a flat list of awards that reference a project by slug, and the
// programs. All three are typed here because all three are in the file - a
// section is read whole or it is not read - but only the first two reach a
// screen. Programs are deliberately off this surface, along with skills, so
// they are parsed and never rendered.
//
// The awards list is flat and keyed to a project by slug rather than nested
// under the project it was won for, which is what lets the awards section read
// it in the order the pack wrote it while a project page reads only its own.
// AwardsFor is that second reading, and it lives here so both directions agree
// on what "this project's awards" means.

// Project is one entry of the projects list: what was built, what it does, and
// the facts that back it up.
type Project struct {
	Slug    string `yaml:"slug"`
	Name    string `yaml:"name"`
	Summary string `yaml:"summary"`

	Highlights []string   `yaml:"highlights"`
	Stack      []string   `yaml:"stack"`
	Teammates  []string   `yaml:"teammates"`
	Links      NamedLinks `yaml:"links"`
	Notes      []string   `yaml:"notes"`

	BuiltAt string `yaml:"built_at"` // the event it was built at, where it was one
	Date    string `yaml:"date"`
}

// Award is one entry of the awards list: a placing, at an event, for the
// project carrying the slug it names.
//
// Placement and Selection are alternatives, not a pair: an award either placed
// or it was selected out of a field, and an entry that carries neither is one
// the pack recorded for the event alone. Which word a surface says for that
// case is the surface's to choose, not the pack's.
type Award struct {
	Slug    string `yaml:"slug"`
	Project string `yaml:"project"` // the slug of the project it was won for
	Event   string `yaml:"event"`
	Year    int    `yaml:"year"`

	Placement string `yaml:"placement"`
	Selection string `yaml:"selection"`
	Track     string `yaml:"track"`

	// Participants is the size of the field the placing was won against, as the
	// pack wrote it: it is "120 teams" or "~1300" as often as it is a number,
	// so it is a string and never arithmetic.
	Participants string `yaml:"participants"`
	Prize        string `yaml:"prize"`

	Extras []string `yaml:"extras"`
	Notes  []string `yaml:"notes"`
}

// Program is one entry of the programs list: a cohort, a mixer, an invitation.
//
// Nothing renders it. It is typed so that the section it shares a file with can
// be parsed at all, and so that the day a surface is asked for it, the fact is
// already read rather than reached for through raw YAML.
type Program struct {
	Slug   string `yaml:"slug"`
	Name   string `yaml:"name"`
	Detail string `yaml:"detail"`
	Date   string `yaml:"date"`
	Year   int    `yaml:"year"`
}

// Projects is the whole parsed projects.yaml: its three lists, each in the
// order the pack wrote it.
type Projects struct {
	Projects []Project `yaml:"projects"`
	Awards   []Award   `yaml:"awards"`
	Programs []Program `yaml:"programs"`
}

// AwardsFor returns the awards won for the project carrying a slug, in the
// order the pack lists them.
//
// This is the lookup the projects section reads a project page's awards with,
// and the one the awards section reads back the other way - so the answer to
// "which awards belong to this project" is written once and cannot drift
// between the two screens that ask it.
func (p *Pack) AwardsFor(slug string) []Award {
	if slug == "" {
		return nil
	}
	var out []Award
	for _, award := range p.Awards {
		if award.Project == slug {
			out = append(out, award)
		}
	}
	return out
}

// parseProjects parses raw projects.yaml bytes into its three lists. Like the
// section parsers beside it, it is factored out of Load so tests can run
// against a fixture rather than the fetched pack.
func parseProjects(b []byte) (Projects, error) {
	var doc Projects
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return Projects{}, fmt.Errorf("parse projects section: %w", err)
	}
	return doc, nil
}
