package data

import (
	"errors"
	"strings"
	"testing"
)

// A connection after the first that cannot open as busy is the hidden
// journal_mode only when DB_URL has a _pragma, in any escape of its key;
// without one it is the plain sentence that names the connection.
func TestBusyOpenNamesJournalOnlyWithAPragma(t *testing.T) {
	busy := errors.New("database is locked")
	for url, journal := range map[string]bool{
		"file:data/app.db":                                         false,
		"file:data/app.db?_busy_timeout=9000":                      false,
		"file:data/app.db?_pragma=cache_size('--');pragma+x=1":     true,
		"file:data/app.db?%5Fpragma=cache_size(1)":                 true,
		"file:data/app.db?_txlock=immediate&_pragma=cache_size(1)": true,
	} {
		err := openProblem(1, busy, url)
		if got := strings.Contains(err.Error(), "_pragma sets journal_mode"); got != journal {
			t.Errorf("%s: %v", url, err)
		}
		if !journal && err.Error() != "connection 2 to the database cannot be opened (database is locked); check the settings in DB_URL." {
			t.Errorf("%s: %v", url, err)
		}
	}
	if err := openProblem(0, busy, "file:x?_pragma=cache_size(1)"); strings.Contains(err.Error(), "journal_mode") {
		t.Errorf("the first connection: %v", err)
	}
}
