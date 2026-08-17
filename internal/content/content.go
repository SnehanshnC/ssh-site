// Package content loads the SSH site's content pack: the shared, fetched
// source of truth for every fact rendered by this surface. Nothing about
// Snehanshn is hardcoded here - see internal/content/pack/README.md and
// scripts/fetch-pack.sh for where the data comes from.
package content

import (
	"embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed pack/*.yaml
var packFS embed.FS

// sections lists the content pack files fetched by scripts/fetch-pack.sh,
// keyed by their name without the .yaml extension.
var sections = []string{"identity", "work", "projects", "links", "hobbies"}

// Pack is the parsed content pack consumed by the TUI. Identity is typed and
// ready to use; the remaining sections are kept as raw YAML bytes for later
// build issues to parse as their own shapes take form.
type Pack struct {
	Identity Identity

	raw map[string][]byte
}

// Identity is the person's identity section of the content pack: their name,
// role, education, and the tagline pool surfaces choose from.
type Identity struct {
	Name      string    `yaml:"name"`
	Role      Role      `yaml:"role"`
	Education Education `yaml:"education"`
	Taglines  []string  `yaml:"taglines"`
	Skills    Skills    `yaml:"skills"`
}

// Role is the job title, company, and program under which the role is held.
type Role struct {
	Title   string `yaml:"title"`
	Company string `yaml:"company"`
	Program string `yaml:"program"`
}

// Education describes a single institution, degree, and enrollment span.
type Education struct {
	Institution string `yaml:"institution"`
	Degree      string `yaml:"degree"`
	StartYear   int    `yaml:"start_year"`
	EndYear     int    `yaml:"end_year"`
}

// Skills groups the skill lists identity.yaml carries.
type Skills struct {
	Languages  []string `yaml:"languages"`
	Frameworks []string `yaml:"frameworks"`
	Tools      []string `yaml:"tools"`
}

// Load reads the embedded content pack, parses identity.yaml into typed
// structs, and keeps the remaining sections available as raw bytes via Raw.
//
// Load will fail to even compile into a binary until scripts/fetch-pack.sh
// (via `make content`) has populated internal/content/pack - that's
// intentional, so a build can never silently ship without real content.
func Load() (*Pack, error) {
	raw := make(map[string][]byte, len(sections))
	for _, section := range sections {
		b, err := packFS.ReadFile("pack/" + section + ".yaml")
		if err != nil {
			return nil, fmt.Errorf("read %s section: %w", section, err)
		}
		raw[section] = b
	}

	identity, err := parseIdentity(raw["identity"])
	if err != nil {
		return nil, err
	}

	return &Pack{Identity: identity, raw: raw}, nil
}

// parseIdentity parses raw identity.yaml bytes into an Identity. It is
// factored out of Load so tests can exercise the parsing logic against a
// hand-written testdata fixture, without depending on the fetched pack.
func parseIdentity(b []byte) (Identity, error) {
	var identity Identity
	if err := yaml.Unmarshal(b, &identity); err != nil {
		return Identity{}, fmt.Errorf("parse identity section: %w", err)
	}
	return identity, nil
}

// Raw returns the unparsed YAML bytes for the named section (identity, work,
// projects, links, or hobbies). It returns nil for an unknown section.
func (p *Pack) Raw(section string) []byte {
	return p.raw[section]
}
