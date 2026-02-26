package engine

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
)

// LogEntry represents a single parsed access log entry.
type LogEntry struct {
	Time   string // HH:MM:SS
	Domain string // destination domain or IP
	Route  string // "proxy" or "direct"
}

// logRe parses Xray access log lines like:
// 2026/02/26 22:38:48.761671 from 127.0.0.1:58550 accepted //docs.google.com:443 [http-in >> proxy]
var logRe = regexp.MustCompile(
	`\d{4}/\d{2}/\d{2}\s+(\d{2}:\d{2}:\d{2})\.\d+\s+from\s+\S+\s+accepted\s+//([^:]+):\d+\s+\[(\S+)\s+>>\s+(\S+)\]`,
)

const maxLogEntries = 200

// LogCapture captures Xray access logs via an os.Pipe and parses them
// into structured entries stored in a ring buffer.
type LogCapture struct {
	mu      sync.Mutex
	entries []LogEntry
	reader  *os.File
	writer  *os.File
	done    chan struct{}
}

// NewLogCapture creates a new log capture backed by an os.Pipe.
// It returns the capture instance and the file path to pass to Xray's
// access log config (e.g., "/dev/fd/N").
func NewLogCapture() (*LogCapture, string, error) {
	r, w, err := os.Pipe()
	if err != nil {
		return nil, "", fmt.Errorf("creating log pipe: %w", err)
	}

	lc := &LogCapture{
		reader: r,
		writer: w,
		done:   make(chan struct{}),
	}

	path := fmt.Sprintf("/dev/fd/%d", w.Fd())

	go lc.readLoop()

	return lc, path, nil
}

// readLoop reads lines from the pipe and parses them into LogEntry values.
func (lc *LogCapture) readLoop() {
	defer close(lc.done)
	scanner := bufio.NewScanner(lc.reader)
	for scanner.Scan() {
		line := scanner.Text()
		if entry, ok := parseLine(line); ok {
			lc.mu.Lock()
			lc.entries = append(lc.entries, entry)
			if len(lc.entries) > maxLogEntries {
				lc.entries = lc.entries[len(lc.entries)-maxLogEntries:]
			}
			lc.mu.Unlock()
		}
	}
}

// parseLine extracts a LogEntry from a raw Xray access log line.
func parseLine(line string) (LogEntry, bool) {
	m := logRe.FindStringSubmatch(line)
	if m == nil {
		return LogEntry{}, false
	}

	route := strings.TrimSuffix(m[4], "]")

	return LogEntry{
		Time:   m[1],
		Domain: m[2],
		Route:  route,
	}, true
}

// Entries returns the last n log entries. If n <= 0, all entries are returned.
func (lc *LogCapture) Entries(n int) []LogEntry {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if n <= 0 || n >= len(lc.entries) {
		out := make([]LogEntry, len(lc.entries))
		copy(out, lc.entries)
		return out
	}

	start := len(lc.entries) - n
	out := make([]LogEntry, n)
	copy(out, lc.entries[start:])
	return out
}

// Close shuts down the log capture, closing both ends of the pipe.
func (lc *LogCapture) Close() {
	lc.writer.Close()
	lc.reader.Close()
	<-lc.done
}
