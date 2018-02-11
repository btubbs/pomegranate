package pomegranate

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetForwardMigrations(t *testing.T) {
	tt := []struct {
		desc         string
		historynames []string
		staticnames  []string
		toRun        []string
		err          error
	}{
		{
			desc:         "run them all",
			historynames: []string{},
			staticnames:  []string{"a", "b", "c", "d"},
			toRun:        []string{"a", "b", "c", "d"},
			err:          nil,
		},
		{
			desc:         "run a couple",
			historynames: []string{"a", "b"},
			staticnames:  []string{"a", "b", "c", "d"},
			toRun:        []string{"c", "d"},
			err:          nil,
		},
		{
			desc:         "nothing to run",
			historynames: []string{"a", "b", "c", "d"},
			staticnames:  []string{"a", "b", "c", "d"},
			toRun:        []string{},
			err:          nil,
		},
		{
			desc:         "too much history",
			historynames: []string{"a", "b", "c"},
			staticnames:  []string{"a", "b"},
			toRun:        nil,
			err:          errors.New("migration history longer than static list"),
		},
		{
			desc:         "mismatched history",
			historynames: []string{"a", "b", "c", "d"},
			staticnames:  []string{"a", "b", "banana", "d"},
			toRun:        nil,
			err: errors.New(
				"migration 3 from history (c) does not match name from static list (banana)"),
		},
	}
	for _, tc := range tt {
		history := namesToHistory(tc.historynames)
		migs := namesToMigs(tc.staticnames)
		toRun, err := getForwardMigrations(history, migs)
		assert.Equal(t, tc.err, err)
		assert.Equal(t, tc.toRun, migsToNames(toRun))
	}
}

func namesToHistory(names []string) []MigrationRecord {
	migs := []MigrationRecord{}
	for _, name := range names {
		migs = append(migs, MigrationRecord{Name: name})
	}
	return migs
}

func namesToMigs(names []string) []Migration {
	migs := []Migration{}
	for _, name := range names {
		migs = append(migs, Migration{Name: name})
	}
	return migs
}

func migsToNames(migs []Migration) []string {
	if migs == nil {
		return nil
	}
	names := []string{}
	for _, mig := range migs {
		names = append(names, mig.Name)
	}
	return names
}
