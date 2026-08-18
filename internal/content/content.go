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

// Pack is the parsed content pack consumed by the TUI. A section is typed by
// the build slice that builds its page; the sections whose slices have not
// landed yet are kept as raw YAML bytes until they do.
type Pack struct {
	Identity Identity
	Links    []Link
	Work     []Job

	// The three lists of the projects section, kept flat here rather than
	// nested in a Projects value so that a consumer reads pack.Awards the same
	// way it reads pack.Work. Programs is parsed and rendered nowhere - see
	// projects.go for why it is typed at all.
	Projects []Project
	Awards   []Award
	Programs []Program

	raw map[string][]byte
}

// Link is one entry of the pack's links section: a permanent slug, a display
// label, and the URL. Surfaces key their display mappings to the slug, never to
// the label, so rewording a link breaks nothing.
type Link struct {
	Slug  string `yaml:"slug"`
	Label string `yaml:"label"`
	URL   string `yaml:"url"`
}

// Link returns the link carrying the given slug.
func (p *Pack) Link(slug string) (Link, bool) {
	for _, link := range p.Links {
		if link.Slug == slug {
			return link, true
		}
	}
	return Link{}, false
}

// NamedLinks is the links block an entry of another section carries: that
// entry's own links, in the order the pack wrote them.
//
// The pack writes the block as a mapping keyed by a permanent name, and a name
// holds either one link or a list of them - fraxai has two repos under `code`.
// Reading both shapes is done here, once, rather than at each of the section
// slices that meets one. The key is the slug, and a link that carries no label
// of its own is shown under its key, so a name is always something to read.
type NamedLinks []Link

// UnmarshalYAML walks the mapping in the order the document wrote it, which is
// the order the pages show; decoding into a Go map would lose it.
func (l *NamedLinks) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("links at line %d: want a mapping of names to links", node.Line)
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		slug, value := node.Content[i].Value, node.Content[i+1]
		items := []*yaml.Node{value}
		if value.Kind == yaml.SequenceNode {
			items = value.Content
		}
		for _, item := range items {
			var link Link
			if err := item.Decode(&link); err != nil {
				return fmt.Errorf("link %s: %w", slug, err)
			}
			link.Slug = slug
			if link.Label == "" {
				link.Label = slug
			}
			*l = append(*l, link)
		}
	}
	return nil
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

	links, err := parseLinks(raw["links"])
	if err != nil {
		return nil, err
	}

	work, err := parseWork(raw["work"])
	if err != nil {
		return nil, err
	}

	projects, err := parseProjects(raw["projects"])
	if err != nil {
		return nil, err
	}

	return &Pack{
		Identity: identity,
		Links:    links,
		Work:     work,
		Projects: projects.Projects,
		Awards:   projects.Awards,
		Programs: projects.Programs,
		raw:      raw,
	}, nil
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

// parseLinks parses raw links.yaml bytes into a slice of Links. Like
// parseIdentity it is factored out of Load so tests can run against a fixture
// rather than the fetched pack.
func parseLinks(b []byte) ([]Link, error) {
	var doc struct {
		Links []Link `yaml:"links"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("parse links section: %w", err)
	}
	return doc.Links, nil
}

// Raw returns the unparsed YAML bytes for the named section (identity, work,
// projects, links, or hobbies), for the sections whose own slice has not typed
// them yet. It returns nil for an unknown section.
func (p *Pack) Raw(section string) []byte {
	return p.raw[section]
}
