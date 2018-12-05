// Package pomegranate implements helper functions and a CLI (pmg) for creating
// and safely running SQL migrations.
//
// Go projects can use pomegranate to ingest .sql files into a Go binary,
// and run them from there.  See the README.md on Github for examples and
// explanations.
package pomegranate

import (
	"time"
)

// MigrationRecord provides information on which migrations are currently in effect.  An array of
// MigrationRecords is referred to as a "state" throughout the Pomegranate source.  These are
// treated as a stack; MigrationRecords are added (inserted into the DB) as migrations run forward,
// and popped off (deleted from the DB) as migrations are run backward.
type MigrationRecord struct {
	Name string    `db:"name"`
	Time time.Time `db:"time"`
	Who  string    `db:"who"`
}

// Migration contains the name and SQL for a migration.  Arrays of Migrations
// are passed between many functions in the Pomegranate source.
// SeperateForwardStatements runs SQL statements seperately, delinieated by ";"
type Migration struct {
	Name        string
	ForwardSQL  []string
	BackwardSQL []string
}

//QuotedForward returns the ForwardSQL field of the Migration, surrounded with
//backticks for easy injection into a migrations.go template.
func (m Migration) QuotedForward() []string {
	fwdSQLArr := []string{}
	for _, sql := range m.ForwardSQL {
		fwdSQLArr = append(fwdSQLArr, "`"+sql+"`")
	}
	return fwdSQLArr
}

// QuotedBackward returns the BackwardSQL field of the Migration, surrounded with
// backticks for easy injection into a migrations.go template.
func (m Migration) QuotedBackward() []string {
	bwdSQLArr := []string{}
	for _, sql := range m.BackwardSQL {
		bwdSQLArr = append(bwdSQLArr, "`"+sql+"`")
	}
	return bwdSQLArr
}

// MigrationLogRecord represents a specific migration run at a specific point in time.  Unlike
// MigrationRecord, this is an append-only table, showing the complete history of all forward and
// backward migrations.  It is populated automatically by a Postgres trigger created in the init
// migration.
type MigrationLogRecord struct {
	ID   int       `db:"id"`
	Time time.Time `db:"time"`
	Name string    `db:"name"`
	Op   string    `db:"op"`
	Who  string    `db:"who"`
}
