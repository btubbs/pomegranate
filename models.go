// Package pomegranate implements helper functions and a CLI (pmg) for creating
// and safely running SQL migrations.
//
// Go projects can use pomegranate to ingest .sql files into a Go binary,
// and run them from there.  See the README.md on Github for examples and
// explanations.
package pomegranate

import "time"

// MigrationRecord provides information on which migrations ran, and when.
// An array of MigrationRecords is referred to as a "state" throughout the
// Pomegranate source.
type MigrationRecord struct {
	Name string    `db:"name"`
	Time time.Time `db:"time"`
	Who  string    `db:"who"`
}

// Migration contains the name and SQL for a migration.  Arrays of Migrations
// are passed between many functions in the Pomegranate source.
type Migration struct {
	Name        string
	ForwardSQL  string
	BackwardSQL string
}

// QuotedForward returns the ForwardSQL field of the Migration, surrounded with
// backticks for easy injection into a migrations.go template.
func (m Migration) QuotedForward() string {
	return "`" + m.ForwardSQL + "`"
}

// QuotedBackward returns the BackwardSQL field of the Migration, surrounded with
// backticks for easy injection into a migrations.go template.
func (m Migration) QuotedBackward() string {
	return "`" + m.BackwardSQL + "`"
}
