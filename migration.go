package pomegranate

import "time"

type Migration struct {
	Name        string
	ForwardSql  string
	BackwardSql string
}

type PastMigration struct {
	Name string    `db:"name"`
	Time time.Time `db:"time"`
	Who  string    `db:"who"`
}
