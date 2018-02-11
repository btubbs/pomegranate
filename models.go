package pomegranate

import "time"

type MigrationRecord struct {
	Name string    `db:"name"`
	Time time.Time `db:"time"`
	Who  string    `db:"who"`
}

type Migration struct {
	Name        string
	ForwardSQL  string
	BackwardSQL string
}

func (m Migration) QuotedForward() string {
	return "`" + m.ForwardSQL + "`"
}

func (m Migration) QuotedBackward() string {
	return "`" + m.BackwardSQL + "`"
}
