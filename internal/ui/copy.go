package ui

import (
	"strings"
	"unicode"

	"github.com/SnehanshnC/ssh-site/internal/ansi"
	"github.com/SnehanshnC/ssh-site/internal/content"
)

// Everything on the card except the art is composed here, from the pack, at
// render time. Nothing about Snehanshn is written in this file: what it holds
// is shortening rules, and each rule is applied through a ladder that offers
// the fullest form first and falls back only when the fuller one does not fit
// the column it has been given.
//
// That ladder is also how the card keeps its promise about links. A truncated
// URL is a broken link, so a link is never cut to fit - it either has a form
// that fits or the layout gives up on this width and restacks.

// cardCopy is the right-hand column of the two-column card, already fitted to
// a width. The first three are plain text and the caller paints them, because
// each is one colour; the link rows arrive painted because each carries two, a
// dim label and the address beside it.
type cardCopy struct {
	role   string
	school string
	quest  []string
	links  []string
}

// linkSlugs are the links the card carries, in the order it carries them.
// Slugs, never labels, so rewording a link in the pack breaks nothing.
var linkSlugs = []string{"github", "linkedin"}

// labelWidth is the card's link label column, as signed off - `github` and
// `linkedin` truncated to a common six columns. A pack whose slugs are shorter
// than that shortens the column with them.
const labelWidth = 6

// composeCopy fits the card's copy into width columns, reporting whether every
// line found a form that fits.
func composeCopy(pack *content.Pack, width int) (cardCopy, bool) {
	role, ok := pick(roleForms(pack.Identity), width)
	if !ok {
		return cardCopy{}, false
	}
	school, ok := pick(schoolForms(pack.Identity), width)
	if !ok {
		return cardCopy{}, false
	}
	links, ok := linkRows(pack, width)
	if !ok {
		return cardCopy{}, false
	}
	quest, ok := questLines(pack.Identity, width)
	if !ok {
		return cardCopy{}, false
	}
	return cardCopy{role: role, school: school, quest: quest, links: links}, true
}

// pick returns the first form that fits, forms being ordered fullest first.
func pick(forms []string, width int) (string, bool) {
	for _, form := range forms {
		if form != "" && ansi.Width(form) <= width {
			return form, true
		}
	}
	return "", false
}

// roleForms ladders the role line down from every fact it carries to the title
// alone: "AI Engineer @ NovaFlow (Y Combinator S25)" to "AI Engineer".
func roleForms(id content.Identity) []string {
	head := id.Role.Title
	if id.Role.Company != "" {
		head += " @ " + id.Role.Company
	}
	if id.Role.Program == "" {
		return []string{head, id.Role.Title}
	}
	return []string{
		head + " (" + id.Role.Program + ")",
		head + " (" + initialise(id.Role.Program) + ")",
		head,
		id.Role.Title,
	}
}

// schoolForms ladders the education line: the degree's fields of study joined
// with the institution, shortening the fields to their initials and the
// institution to its distinguishing word before it gives either up.
func schoolForms(id content.Identity) []string {
	inst := shortInstitution(id.Education.Institution)
	fields := degreeFields(id.Education.Degree)
	full := strings.Join(fields, " and ")
	short := make([]string, len(fields))
	for i, f := range fields {
		short[i] = initialise(f)
	}
	abbrev := strings.Join(short, " + ")

	// The institution outlives the detail of the degree: shorten the fields to
	// their initials before giving up the school, and give up the fields
	// entirely before giving up the school's name.
	forms := []string{}
	if inst != "" {
		for _, subject := range []string{full, abbrev} {
			if subject != "" {
				forms = append(forms, subject+" @ "+inst)
			}
		}
	}
	return append(forms, full, abbrev, inst)
}

// questLines splits the pack's leading tagline at its own colon, which is the
// shape the card was signed off around: a dim-cyan label row and the quest
// under it. A tagline with no colon is one row.
func questLines(id content.Identity, width int) ([]string, bool) {
	if len(id.Taglines) == 0 {
		return nil, true
	}
	tagline := id.Taglines[0]
	head, tail, split := strings.Cut(tagline, ":")
	if !split {
		if ansi.Width(tagline) > width {
			return nil, false
		}
		return []string{tagline}, true
	}
	head, tail = strings.TrimSpace(head)+":", strings.TrimSpace(tail)
	if ansi.Width(head) > width || ansi.Width(tail) > width {
		return nil, false
	}
	return []string{head, tail}, true
}

// linkRows fits the card's links into width columns.
//
// The label column is all-or-nothing across the block, so the two rows stay
// aligned; the URL form is chosen per row, so a short link keeps its host while
// a long one gives it up. Nothing is ever truncated: if no form of some link
// fits, the caller restacks.
func linkRows(pack *content.Pack, width int) ([]string, bool) {
	links := make([]content.Link, 0, len(linkSlugs))
	label := labelWidth
	for _, slug := range linkSlugs {
		link, ok := pack.Link(slug)
		if !ok {
			continue
		}
		links = append(links, link)
		if len(slug) < label {
			label = len(slug)
		}
	}
	if len(links) == 0 {
		return nil, true
	}

	for _, labelled := range []bool{true, false} {
		budget, prefix := width, 0
		if labelled {
			prefix = label + 1
			budget = width - prefix
		}
		rows := make([]string, 0, len(links))
		for _, link := range links {
			shown, ok := pick(urlForms(link.URL), budget)
			if !ok {
				rows = nil
				break
			}
			row := paint(textState, shown)
			if labelled {
				row = paint(dimState, truncate(link.Slug, label)+" ") + row
			}
			rows = append(rows, row)
		}
		if rows != nil {
			return rows, true
		}
	}
	return nil, false
}

// urlForms ladders one URL down from host and path to the last path segment.
// Every form is still a complete, navigable address given the label beside it -
// none of them is a prefix of the URL cut off partway.
func urlForms(raw string) []string {
	trimmed := strings.TrimSuffix(strings.TrimPrefix(
		strings.TrimPrefix(raw, "https://"), "http://"), "/")
	trimmed = strings.TrimPrefix(trimmed, "www.")

	forms := []string{trimmed}
	if _, path, ok := strings.Cut(trimmed, "/"); ok && path != "" {
		forms = append(forms, path)
		if i := strings.LastIndex(path, "/"); i >= 0 && i+1 < len(path) {
			forms = append(forms, path[i+1:])
		}
	}
	return forms
}

// initialise shortens a phrase the way a person would say it out loud: a run of
// two or more words collapses to its initials, a word standing on its own keeps
// its name, and any token carrying a digit is already a code and is kept whole.
// "Y Combinator S25" becomes "YC S25" and "Computer Science" becomes "CS", but
// "Mathematics" stays "Mathematics" rather than becoming "M".
func initialise(phrase string) string {
	var out []string
	var run []string
	flush := func() {
		switch len(run) {
		case 0:
		case 1:
			out = append(out, run[0])
		default:
			var initials strings.Builder
			for _, word := range run {
				initials.WriteRune(unicode.ToUpper([]rune(word)[0]))
			}
			out = append(out, initials.String())
		}
		run = nil
	}
	for _, word := range strings.Fields(phrase) {
		if hasDigit(word) {
			flush()
			out = append(out, word)
			continue
		}
		run = append(run, word)
	}
	flush()
	return strings.Join(out, " ")
}

// degreeFields pulls the fields of study out of a degree, dropping the award
// that precedes them: "BS in Computer Science and Mathematics" gives Computer
// Science and Mathematics.
func degreeFields(degree string) []string {
	if _, subject, ok := strings.Cut(degree, " in "); ok {
		degree = subject
	}
	var fields []string
	for _, part := range strings.Split(degree, " and ") {
		for _, field := range strings.Split(part, ",") {
			if f := strings.TrimSpace(field); f != "" {
				fields = append(fields, f)
			}
		}
	}
	return fields
}

// genericInstitutionWords are the words an institution's name shares with every
// other institution, and so the ones that carry none of its identity.
var genericInstitutionWords = map[string]bool{
	"university": true, "college": true, "institute": true, "school": true,
	"academy": true, "polytechnic": true, "of": true, "the": true, "at": true,
}

// shortInstitution keeps the distinguishing part of an institution's name:
// "Rutgers University - New Brunswick" becomes "Rutgers". The campus qualifier
// goes first, then the generic words, and it stops before it would return
// nothing.
func shortInstitution(name string) string {
	if head, _, ok := strings.Cut(name, " - "); ok {
		name = head
	}
	if head, _, ok := strings.Cut(name, ","); ok {
		name = head
	}
	words := strings.Fields(name)
	kept := words[:0]
	for _, word := range words {
		if !genericInstitutionWords[strings.ToLower(word)] {
			kept = append(kept, word)
		}
	}
	if len(kept) == 0 {
		return strings.TrimSpace(name)
	}
	return strings.Join(kept, " ")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func hasDigit(s string) bool {
	for _, r := range s {
		if unicode.IsDigit(r) {
			return true
		}
	}
	return false
}
