package ui

import (
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/SnehanshnC/ssh-site/internal/content"
)

// The stub section page.
//
// The navigation shell built the stack, not the sections: the real links and
// hobbies pages are two slices of their own, each of which types its own part
// of the content pack and designs its own screen. What is here is the page
// shape those two replace - a list that drills into a detail - reading the pack
// generically so that it shows real facts rather than invented ones while it
// stands in for them.
//
// Work, projects and awards have already gone: openSection routes `w`, `p` and
// `a` to the typed pages in work.go, projects.go and awards.go and this file
// lost its rows for them, which is the trade each section slice makes. When the
// last of the two lands, this file goes with it - the generic pack reading
// above all, which exists only because a section that no slice has typed yet
// has no shape to read it into.

// section is one destination the card's nav row and the letter jumps open.
type section struct {
	key  string // the letter that jumps here
	name string // the header, the breadcrumb, and the jump's name in the hints
	file string // the pack section its entries are read from
	list string // the key inside that file
	one  string // what one entry is called, for the count beside the header
	many string // and what more than one is called, which is not always one+s
}

var stubSections = []section{
	{"l", "links", "links", "links", "link", "links"},
	{"h", "hobbies", "hobbies", "hobbies", "hobby", "hobbies"},
}

// The fields an entry is named and described by, best first. Sections do not
// share a schema, so the page takes the first field an entry actually carries
// rather than asking each section to declare one.
var (
	nameFields = []string{"company", "name", "event", "label", "title", "slug"}
	// A URL is deliberately not here. The list row shortens its description to
	// fit the column, and a shortened URL is a dead link, so links describe
	// themselves on their own page where the address is wrapped whole.
	aboutFields = []string{"role", "summary", "placement", "selection", "detail", "blurb"}
)

// openStub returns the stand-in page for a section no slice has typed yet.
func openStub(pack *content.Pack, key string) Page {
	for _, sec := range stubSections {
		if sec.key == key {
			return listPage{section: sec, entries: readEntries(pack, sec)}
		}
	}
	return nil
}

// listPage is a section's index: one selectable row per entry, drilling into
// that entry's own page.
type listPage struct {
	section section
	entries []entry
}

func (p listPage) Chrome() Chrome {
	return Chrome{
		Title:      strings.ToUpper(p.section.name),
		Suffix:     count(len(p.entries), p.section.one, p.section.many),
		Crumbs:     []string{p.section.name},
		Selectable: true,
	}
}

func (p listPage) Key(key string, cursor int) (Action, Page) {
	if key == "enter" && cursor >= 0 && cursor < len(p.entries) {
		return Push, detailPage{section: p.section, entry: p.entries[cursor]}
	}
	return Ignored, nil
}

// Blocks lays each entry out as one row, in the shared list idiom. The name
// column exists to line the descriptions up under each other, so a section with
// nothing to say beside its names asks for no column at all.
func (p listPage) Blocks(width, cursor int) [][]string {
	names, described := make([]string, len(p.entries)), false
	for i, e := range p.entries {
		names[i] = e.first(nameFields)
		described = described || e.first(aboutFields) != ""
	}
	column := 0
	if described {
		column = nameColumn(names, width)
	}

	blocks := make([][]string, 0, len(p.entries))
	for i, e := range p.entries {
		blocks = append(blocks, []string{
			listRow(width, i == cursor, names[i], e.first(aboutFields), column),
			"",
		})
	}
	return blocks
}

// detailPage is one entry, as the pack wrote it: a scrolling document rather
// than a list, so it is the shape that proves scrolling as the list proves the
// cursor.
type detailPage struct {
	section section
	entry   entry
}

func (p detailPage) Chrome() Chrome {
	crumbs := []string{p.section.name}
	if slug := p.entry.value("slug"); slug != "" {
		crumbs = append(crumbs, slug)
	}
	return Chrome{Title: p.entry.first(nameFields), Suffix: p.section.one, Crumbs: crumbs}
}

func (p detailPage) Key(string, int) (Action, Page) { return Ignored, nil }

func (p detailPage) Blocks(width, _ int) [][]string {
	var blocks [][]string
	for _, f := range p.entry.fields {
		if f.name == "slug" {
			continue // the breadcrumb already carries it
		}
		rows := []string{paint(dimState, f.name)}
		if f.value != "" {
			rows = append(rows, indentRows(
				wrap(paint(textState, f.value), max(width-bodyIndent, 1)), bodyIndent)...)
		}
		for _, item := range f.values {
			rows = append(rows, bullets(item, width)...)
		}
		blocks = append(blocks, append(rows, ""))
	}
	return blocks
}

// --- reading the pack generically ---

// entry is one item of a pack section, kept in the order the pack wrote it: an
// ordered list of fields and no schema at all.
type entry struct {
	fields []field
}

// field is one of an entry's keys. A scalar carries a value, a sequence of
// scalars carries values, and anything deeper is left out - shaping nested
// structure is the job of the slice that types the section.
type field struct {
	name   string
	value  string
	values []string
}

// value returns a named scalar field, or the empty string.
func (e entry) value(name string) string {
	for _, f := range e.fields {
		if f.name == name {
			return f.value
		}
	}
	return ""
}

// first returns the first of the named fields the entry actually carries.
func (e entry) first(names []string) string {
	for _, name := range names {
		if v := e.value(name); v != "" {
			return v
		}
	}
	return ""
}

func readEntries(pack *content.Pack, sec section) []entry {
	var doc yaml.Node
	if err := yaml.Unmarshal(pack.Raw(sec.file), &doc); err != nil || len(doc.Content) == 0 {
		return nil
	}
	list := mapValue(doc.Content[0], sec.list)
	if list == nil || list.Kind != yaml.SequenceNode {
		return nil
	}
	entries := make([]entry, 0, len(list.Content))
	for _, node := range list.Content {
		entries = append(entries, readEntry(node))
	}
	return entries
}

func mapValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func readEntry(node *yaml.Node) entry {
	var e entry
	if node.Kind != yaml.MappingNode {
		return e
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		name, value := node.Content[i].Value, node.Content[i+1]
		switch value.Kind {
		case yaml.ScalarNode:
			if value.Tag == "!!null" || value.Value == "" {
				continue
			}
			e.fields = append(e.fields, field{name: name, value: value.Value})
		case yaml.SequenceNode:
			var items []string
			for _, item := range value.Content {
				if item.Kind == yaml.ScalarNode && item.Value != "" {
					items = append(items, item.Value)
				}
			}
			if len(items) > 0 {
				e.fields = append(e.fields, field{name: name, values: items})
			}
		}
	}
	return e
}
