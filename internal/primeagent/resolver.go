// Package primeagent resolves Prime Agent sessions for panes herdr cannot
// describe itself. herdr reports agent="prime-agent" from the terminal title
// but has no profile for it, so a pane's `agent_session` is empty and neither
// conversation history nor `agent prompt` can work through herdr alone. The
// authoritative source is the Prime Agent daemon: `prime-agent list --json`
// names every live session together with its transcript path.
package primeagent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	cacheTTL  = 5 * time.Second
	maxOutput = 8 * 1024 * 1024
)

// queryTimeout bounds one `prime-agent list --json`. A variable rather than a
// constant so tests can widen it: the first exec of a freshly written script on
// macOS can take most of a second on its own.
var queryTimeout = 3 * time.Second

var errOutputLimit = errors.New("prime-agent list output limit exceeded")

// Session is one live top-level Prime Agent session.
type Session struct {
	Name string
	ID   string
	File string
	Cwd  string
}

// Resolver shells out to the Prime Agent CLI and caches the answer briefly,
// because the agents poller asks for every pane on every tick.
type Resolver struct {
	binary  string
	mu      sync.Mutex
	cached  []Session
	expires time.Time
}

// NewResolver returns a resolver that uses the `prime-agent` on PATH.
func NewResolver() *Resolver {
	return &Resolver{binary: "prime-agent"}
}

// IsPrime reports whether an agent label herdr produced names Prime Agent.
func IsPrime(agent string) bool {
	switch strings.ToLower(strings.TrimSpace(agent)) {
	case "prime-agent", "primeagent", "prime":
		return true
	default:
		return false
	}
}

// NameFromTitle extracts the session name Prime Agent writes into its terminal
// title, "prime-agent - <name> - <directory>". Titles without that shape yield
// an empty string rather than a guess.
func NameFromTitle(title string) string {
	parts := strings.Split(strings.TrimSpace(title), " - ")
	if len(parts) < 3 || strings.TrimSpace(parts[0]) != "prime-agent" {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// Lookup finds the session for a pane. The name (from the terminal title or
// herdr's agent name) is the primary key; when it is empty or unknown, a cwd
// that exactly one session runs in is accepted as a fallback, so a worker in
// its own worktree still resolves before it has reported a title.
func (r *Resolver) Lookup(ctx context.Context, name, cwd string) (Session, bool) {
	sessions, err := r.Sessions(ctx)
	if err != nil {
		return Session{}, false
	}
	name = strings.TrimSpace(name)
	if name != "" {
		for _, session := range sessions {
			if session.Name == name {
				return session, true
			}
		}
	}
	cwd = strings.TrimSpace(cwd)
	if cwd == "" {
		return Session{}, false
	}
	var match Session
	matches := 0
	for _, session := range sessions {
		if session.Cwd == cwd {
			match = session
			matches++
		}
	}
	if matches != 1 {
		return Session{}, false
	}
	return match, true
}

// Sessions returns the live top-level sessions, from cache when it is fresh.
func (r *Resolver) Sessions(ctx context.Context) ([]Session, error) {
	r.mu.Lock()
	if r.cached != nil && time.Now().Before(r.expires) {
		cached := r.cached
		r.mu.Unlock()
		return cached, nil
	}
	r.mu.Unlock()

	sessions, err := r.query(ctx)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	r.cached = sessions
	r.expires = time.Now().Add(cacheTTL)
	r.mu.Unlock()
	return sessions, nil
}

func (r *Resolver) query(ctx context.Context) ([]Session, error) {
	if _, err := exec.LookPath(r.binary); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, r.binary, "list", "--json")
	var stdout bytes.Buffer
	command.Stdout = &limitedWriter{buffer: &stdout, remaining: maxOutput}
	if err := command.Run(); err != nil {
		return nil, err
	}
	return parseList(stdout.Bytes())
}

func parseList(data []byte) ([]Session, error) {
	var payload struct {
		Sessions []struct {
			Name        string `json:"sessionName"`
			ID          string `json:"sessionId"`
			File        string `json:"sessionFile"`
			Cwd         string `json:"cwd"`
			RuntimeKind string `json:"runtimeKind"`
		} `json:"sessions"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	sessions := make([]Session, 0, len(payload.Sessions))
	for _, entry := range payload.Sessions {
		if entry.RuntimeKind != "top-level" || entry.ID == "" {
			continue
		}
		sessions = append(sessions, Session{
			Name: strings.TrimSpace(entry.Name),
			ID:   entry.ID,
			File: entry.File,
			Cwd:  entry.Cwd,
		})
	}
	return sessions, nil
}

type limitedWriter struct {
	buffer    *bytes.Buffer
	remaining int
}

func (w *limitedWriter) Write(data []byte) (int, error) {
	if len(data) > w.remaining {
		return 0, errOutputLimit
	}
	w.remaining -= len(data)
	return w.buffer.Write(data)
}
