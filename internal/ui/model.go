// Package ui holds the Bubble Tea model rendered to SSH visitors.
package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/SnehanshnC/ssh-site/internal/ansi"
	"github.com/SnehanshnC/ssh-site/internal/content"
)

// Model is the navigation shell: the arrival card, a stack of pages over it,
// and a help overlay over both.
//
// Arrival is the card with a live highlight on its nav row. Enter opens the
// highlighted section as a full-screen page pushed onto the stack; esc and
// backspace pop one level until the card is back. The shape is soft-serve's,
// which is the pattern the feature harvest recommended and the one the
// ticket-06 prototype tried and adopted.
//
// The stack holds the position each page was left in rather than the pages
// holding it themselves. That is what makes a resize survivable: a new window
// size recomposes every body against a new width, and a model that kept the
// cursor inside the body it composed would have to throw the visitor back to
// arrival to stay consistent. Here the size changes and nothing else does.
type Model struct {
	pack   *content.Pack
	width  int
	height int

	stack []frame
	nav   int
	help  bool

	// quitting is set the moment any of q, ctrl+c, a page's own Quit action, or
	// the idle timeout decide the session is over - see quit() in exit.go. It
	// is what View switches on to draw the goodbye line instead of the chrome.
	quitting bool

	// idleGen counts keypresses. Every keypress schedules a fresh idleTick
	// carrying the generation it was scheduled at; a tick whose generation does
	// not match idleGen when it arrives was scheduled before some later
	// keypress and is a stale check on a session that has not, in fact, sat
	// idle - see key() and exit.go.
	idleGen int
}

// sectionKeys are the letters that open a section from anywhere, in the order
// the card's nav row names them and with `h` on the end: hobbies has no seat on
// that row, which was signed off as six items, so the key is its only way in
// and the help overlay is where it is advertised.
var sectionKeys = []string{"w", "p", "a", "l", "h"}

// typedSections maps a letter to the page its own build slice wrote for it.
// Hobbies was the last section left generic, so every letter jump now has a
// page of its own here - section.go's stand-in, which read the pack
// generically for whichever section had not been typed yet, is gone with it.
var typedSections = map[string]func(*content.Pack) Page{
	"w": openWork,
	"p": openProjects,
	"a": openAwards,
	"l": openLinks,
	"h": openHobbies,
}

var jumpKeys = func() map[string]bool {
	keys := make(map[string]bool, len(sectionKeys))
	for _, key := range sectionKeys {
		keys[key] = true
	}
	return keys
}()

// openSection returns the page a letter jump or a nav item opens, or nil where
// the letter opens nothing.
func openSection(pack *content.Pack, key string) Page {
	if open, ok := typedSections[key]; ok {
		return open(pack)
	}
	return nil
}

// New builds a Model from the loaded content pack and the session's initial
// PTY window size.
func New(pack *content.Pack, width, height int) Model {
	return Model{pack: pack, width: width, height: height}
}

// Init implements tea.Model. It starts the idle timer: arrival counts as
// activity in every other sense, so the clock the visitor gets is the full
// ten minutes, timed from the first frame rather than the first keypress.
func (m Model) Init() tea.Cmd {
	return idleTick(m.idleGen)
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resettle()
		return m, nil
	case tea.KeyPressMsg:
		return m.key(msg.String())
	case idleTickMsg:
		if msg.gen != m.idleGen {
			// Some later keypress has already scheduled its own tick; this one
			// is checking on a session that is no longer the one it thinks it
			// is idle-timing.
			return m, nil
		}
		return m.quit()
	}
	return m, nil
}

// key resets the idle timer against this keypress - any keypress counts, so
// the reset happens here rather than deeper in route, which only some keys
// reach - then routes the key and reschedules the next idle check, unless
// routing just ended the session, in which case there is no session left to
// time.
func (m Model) key(key string) (tea.Model, tea.Cmd) {
	m.idleGen++
	next, cmd := m.route(key)
	if next.(Model).quitting {
		return next, cmd
	}
	return next, tea.Batch(cmd, idleTick(m.idleGen))
}

// route sends one keypress to whatever it means.
//
// The shell claims a key before any page sees it only where it has made a
// promise that holds everywhere: quit, help, and the letter jumps. Everything
// else is offered to the open page first, and a page that returns Ignored hands
// it back for the shell's own default.
func (m Model) route(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "q", "ctrl+c":
		return m.quit()
	}
	if m.help {
		if helpKeys[key] {
			m.help = false
		}
		return m, nil
	}
	switch {
	case key == "?":
		m.help = true
		return m, nil
	case jumpKeys[key]:
		return m.jump(key), nil
	case len(m.stack) == 0:
		return m.cardKey(key)
	default:
		return m.pageKey(key)
	}
}

// jump opens a section from anywhere, replacing the stack rather than growing
// it: the letters are a way to change subject, not a way to go deeper, so
// jumping to work from a project's page leaves one level to pop, not two.
func (m Model) jump(key string) Model {
	page := openSection(m.pack, key)
	if page == nil {
		return m
	}
	m.stack = []frame{{page: page}}
	m.resettle()
	return m
}

// cardKey moves the highlight along the card's nav row and opens what it lands
// on. The row it moves along is the one the card actually drew at this size,
// which is five items on the restack and six on the two-column card.
func (m Model) cardKey(key string) (tea.Model, tea.Cmd) {
	items := cardNav(m.pack, m.width, m.height)
	if len(items) == 0 {
		return m, nil
	}
	switch key {
	case "left", "up", "shift+tab":
		m.nav = (m.nav - 1 + len(items)) % len(items)
	case "right", "down", "tab":
		m.nav = (m.nav + 1) % len(items)
	case "home":
		m.nav = 0
	case "end":
		m.nav = len(items) - 1
	case "enter":
		return m.open(items[m.nav])
	}
	return m, nil
}

// open is what enter over a nav item does. `[q] quit` is an item the highlight
// lands on like any other and pressing enter over it ends the session, which is
// the whole point of the page protocol returning an action rather than a bool.
func (m Model) open(item navItem) (tea.Model, tea.Cmd) {
	switch item.key {
	case "q":
		return m.quit()
	case "?":
		m.help = true
		return m, nil
	default:
		return m.jump(item.key), nil
	}
}

// pageKey offers a key to the open page and applies the shell's default to
// whatever the page hands back.
func (m Model) pageKey(key string) (tea.Model, tea.Cmd) {
	m.stack = fork(m.stack)
	top := &m.stack[len(m.stack)-1]
	switch action, next := top.page.Key(key, top.cursor); action {
	case Consumed:
		m.resettle()
		return m, nil
	case Push:
		if next == nil {
			return m, nil
		}
		m.stack = append(m.stack, frame{page: next})
		m.resettle()
		return m, nil
	case Pop:
		return m.pop(), nil
	case Quit:
		return m.quit()
	}

	switch key {
	case "esc", "backspace":
		return m.pop(), nil
	}
	m.scroll(top, key)
	return m, nil
}

// pop goes back one level. Popping the first page is what returns the visitor
// to the card.
func (m Model) pop() Model {
	if len(m.stack) > 0 {
		m.stack = m.stack[:len(m.stack)-1]
	}
	return m
}

// scroll moves the cursor or the scroll offset, by page kind: a list counts in
// items, because that is what the visitor is looking at, and a document counts
// in rows. `space` pages a document down and does nothing to a list, where
// there is no partially-seen thing for it to bring into view.
func (m Model) scroll(f *frame, key string) {
	cols, rows := pageBody(m.width, m.height)
	chrome := f.page.Chrome()
	blocks := f.page.Blocks(cols, f.cursor)

	if chrome.Selectable {
		last := max(len(blocks)-1, 0)
		switch key {
		case "up":
			f.cursor = max(f.cursor-1, 0)
		case "down":
			f.cursor = min(f.cursor+1, last)
		case "pgup":
			f.cursor = max(f.cursor-pageStep, 0)
		case "pgdown":
			f.cursor = min(f.cursor+pageStep, last)
		case "home":
			f.cursor = 0
		case "end":
			f.cursor = last
		}
	} else {
		flat, _ := flatten(blocks)
		bottom := max(len(flat)-rows, 0)
		switch key {
		case "up":
			f.scroll = max(f.scroll-1, 0)
		case "down":
			f.scroll = min(f.scroll+1, bottom)
		case "pgup":
			f.scroll = max(f.scroll-rows, 0)
		case "pgdown", "space":
			f.scroll = min(f.scroll+rows, bottom)
		case "home":
			f.scroll = 0
		case "end":
			f.scroll = bottom
		}
	}
	f.settle(blocks, rows, chrome.Selectable)
}

// fork copies the stack before anything on it moves.
//
// A Model is a value, Bubble Tea hands the same one back on every update, and a
// frame's cursor and scroll offset are written in place - so without this a
// keypress would reach backwards through the shared backing array and move the
// position inside models that have already been rendered.
func fork(stack []frame) []frame {
	return append(make([]frame, 0, len(stack)+1), stack...)
}

// resettle bounds every position the shell holds against the size it now has.
// It bounds and never resets: a window that grew leaves everything where it
// was, and a window that shrank moves only what no longer fits.
func (m *Model) resettle() {
	if items := cardNav(m.pack, m.width, m.height); len(items) > 0 {
		m.nav = min(max(m.nav, 0), len(items)-1)
	}
	cols, rows := pageBody(m.width, m.height)
	m.stack = fork(m.stack)
	for i := range m.stack {
		f := &m.stack[i]
		chrome := f.page.Chrome()
		f.settle(f.page.Blocks(cols, f.cursor), rows, chrome.Selectable)
	}
}

// View implements tea.Model. Every screen but one draws into the alt screen,
// the full-window mode the page stack is specced against; the goodbye line is
// the one exception; see quit() and goodbye() in exit.go for why it draws
// outside it instead.
func (m Model) View() tea.View {
	if m.quitting {
		v := tea.NewView(goodbye(m.pack))
		v.AltScreen = false
		return v
	}
	v := tea.NewView(m.screen())
	v.AltScreen = true
	return v
}

func (m Model) screen() string {
	small := m.width < minCols || m.height < minRows

	var cv *ansi.Canvas
	switch {
	case small:
		cv = ansi.NewCanvas(max(m.width-chromeCol, 0), m.height)
		drawPlea(cv)
	case len(m.stack) == 0:
		cv, _ = fitCard(m.pack, m.width, m.height, m.nav)
	default:
		cv = ansi.NewCanvas(m.width-chromeCol, m.height)
		renderPage(cv, m.stack[len(m.stack)-1], m.width, m.height)
	}
	// The plea is the whole screen: a window with no room for a page has none
	// for a box of keys over it either.
	if m.help && !small {
		drawHelp(cv)
	}
	return cv.Render()
}
