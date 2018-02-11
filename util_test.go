package pomegranate

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfirm(t *testing.T) {
	tt := []struct {
		input string
		err   error
	}{
		{
			input: "y\n",
			err:   nil,
		},
		{
			input: "n\n",
			err:   errors.New("cancelled"),
		},
		{
			input: "banana\n",
			err:   errors.New("Invalid option: banana"),
		},
		{
			input: "y", // no newline!
			err:   errors.New("EOF"),
		},
	}
	for _, tc := range tt {
		err := getConfirm(goodMigrations, "", strings.NewReader(tc.input))
		assert.Equal(t, tc.err, err)
	}
}

func TestNameInHistory(t *testing.T) {
	tt := []struct {
		name    string
		history []MigrationRecord
		result  bool
	}{
		{
			name:    "foo",
			history: []MigrationRecord{MigrationRecord{Name: "foo"}},
			result:  true,
		},
		{
			name:    "foo",
			history: []MigrationRecord{},
			result:  false,
		},
	}
	for _, tc := range tt {
		assert.Equal(t, tc.result, nameInHistory(tc.name, tc.history))
	}
}

func TestNameInMigrations(t *testing.T) {
	tt := []struct {
		name       string
		migrations []Migration
		result     bool
	}{
		{
			name:       "foo",
			migrations: []Migration{Migration{Name: "foo"}},
			result:     true,
		},
		{
			name:       "foo",
			migrations: []Migration{},
			result:     false,
		},
	}
	for _, tc := range tt {
		assert.Equal(t, tc.result, nameInMigrationList(tc.name, tc.migrations))
	}
}

func TestTrimMigrationsTail(t *testing.T) {
	tt := []struct {
		name string
		in   []Migration
		out  []Migration
		err  error
	}{
		{
			name: "foo",
			in: []Migration{
				Migration{Name: "foo"},
				Migration{Name: "bar"},
			},
			out: []Migration{
				Migration{Name: "foo"},
			},
			err: nil,
		},
		{
			name: "banana",
			in: []Migration{
				Migration{Name: "foo"},
				Migration{Name: "bar"},
			},
			out: nil,
			err: errors.New("migration banana not found"),
		},
	}
	for _, tc := range tt {
		res, err := trimMigrationsTail(tc.name, tc.in)
		assert.Equal(t, tc.err, err)
		assert.Equal(t, tc.out, res)
	}
}

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

func TestGetMigrationsToReverse(t *testing.T) {
	tt := []struct {
		desc         string
		name         string
		historynames []string
		staticnames  []string
		out          []string
		err          error
	}{
		{
			desc:         "reverse all",
			name:         "a",
			historynames: []string{"a", "b", "c"},
			staticnames:  []string{"a", "b", "c"},
			out:          []string{"c", "b", "a"},
			err:          nil,
		},
		{
			desc:         "reverse most",
			name:         "a",
			historynames: []string{"a", "b", "c"},
			staticnames:  []string{"a", "b", "c", "d"},
			out:          []string{"c", "b", "a"},
			err:          nil,
		},
		{
			desc:         "reverse a couple",
			name:         "b",
			historynames: []string{"a", "b", "c"},
			staticnames:  []string{"a", "b", "c", "d"},
			out:          []string{"c", "b"},
			err:          nil,
		},
		{
			desc:         "nothing to reverse",
			name:         "d",
			historynames: []string{"a", "b", "c"},
			staticnames:  []string{"a", "b", "c", "d"},
			out:          nil,
			err:          errors.New("migration d not in history"),
		},
		{
			desc:         "trimtail fail",
			name:         "d",
			historynames: []string{"a", "b", "c", "d"},
			staticnames:  []string{"a", "b", "c"},
			out:          nil,
			err:          errors.New("migration d not found"),
		},
		{
			desc:         "weird prehistory",
			name:         "a",
			historynames: []string{"banana", "a", "b", "c"},
			staticnames:  []string{"a", "b", "c"},
			out:          nil,
			err:          errors.New("history in DB has 4 migrations, but we have source for 3 migrations up to and including c"),
		},
		{
			desc:         "mismatched history/static",
			name:         "d",
			historynames: []string{"a", "b", "c"},
			staticnames:  []string{"a", "banana", "c"},
			out:          nil,
			err:          errors.New("migration 2 from history (b) does not match name from static list (banana)"),
		},
	}
	for _, tc := range tt {
		history := namesToHistory(tc.historynames)
		migs := namesToMigs(tc.staticnames)
		out, err := getMigrationsToReverse(tc.name, history, migs)
		assert.Equal(t, migsToNames(out), tc.out)
		assert.Equal(t, err, tc.err)
	}
	// mismatched history
	// name missing from history
}
