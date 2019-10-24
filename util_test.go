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

func TestNameInState(t *testing.T) {
	tt := []struct {
		name   string
		state  []MigrationRecord
		result bool
	}{
		{
			name:   "foo",
			state:  []MigrationRecord{MigrationRecord{Name: "foo"}},
			result: true,
		},
		{
			name:   "foo",
			state:  []MigrationRecord{},
			result: false,
		},
	}
	for _, tc := range tt {
		assert.Equal(t, tc.result, nameInState(tc.name, tc.state))
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
		desc        string
		statenames  []string
		staticnames []string
		toRun       []string
		err         error
	}{
		{
			desc:        "run them all",
			statenames:  []string{},
			staticnames: []string{"a", "b", "c", "d"},
			toRun:       []string{"a", "b", "c", "d"},
			err:         nil,
		},
		{
			desc:        "run a couple",
			statenames:  []string{"a", "b"},
			staticnames: []string{"a", "b", "c", "d"},
			toRun:       []string{"c", "d"},
			err:         nil,
		},
		{
			desc:        "nothing to run",
			statenames:  []string{"a", "b", "c", "d"},
			staticnames: []string{"a", "b", "c", "d"},
			toRun:       []string{},
			err:         nil,
		},
		{
			desc:        "too much state",
			statenames:  []string{"a", "b", "c"},
			staticnames: []string{"a", "b"},
			toRun:       nil,
			err:         errors.New("migration state (3 entries) longer than static list (2 entries)"),
		},
		{
			desc:        "mismatched state",
			statenames:  []string{"a", "b", "c", "d"},
			staticnames: []string{"a", "b", "banana", "d"},
			toRun:       nil,
			err: errors.New(
				"migration 3 from state (c) does not match name from static list (banana)"),
		},
	}
	for _, tc := range tt {
		state := namesToState(tc.statenames)
		migs := namesToMigs(tc.staticnames)
		toRun, err := getForwardMigrations(state, migs)
		assert.Equal(t, tc.err, err)
		assert.Equal(t, tc.toRun, migsToNames(toRun))
	}
}

func TestGetMigrationsToReverse(t *testing.T) {
	tt := []struct {
		desc        string
		name        string
		statenames  []string
		staticnames []string
		out         []string
		err         error
	}{
		{
			desc:        "reverse all",
			name:        "a",
			statenames:  []string{"a", "b", "c"},
			staticnames: []string{"a", "b", "c"},
			out:         []string{"c", "b", "a"},
			err:         nil,
		},
		{
			desc:        "reverse most",
			name:        "a",
			statenames:  []string{"a", "b", "c"},
			staticnames: []string{"a", "b", "c", "d"},
			out:         []string{"c", "b", "a"},
			err:         nil,
		},
		{
			desc:        "reverse a couple",
			name:        "b",
			statenames:  []string{"a", "b", "c"},
			staticnames: []string{"a", "b", "c", "d"},
			out:         []string{"c", "b"},
			err:         nil,
		},
		{
			desc:        "nothing to reverse",
			name:        "d",
			statenames:  []string{"a", "b", "c"},
			staticnames: []string{"a", "b", "c", "d"},
			out:         nil,
			err:         errors.New("migration d not in state"),
		},
		{
			desc:        "trimtail fail",
			name:        "d",
			statenames:  []string{"a", "b", "c", "d"},
			staticnames: []string{"a", "b", "c"},
			out:         nil,
			err:         errors.New("migration d not found"),
		},
		{
			desc:        "weird prestate",
			name:        "a",
			statenames:  []string{"banana", "a", "b", "c"},
			staticnames: []string{"a", "b", "c"},
			out:         nil,
			err:         errors.New("state in DB has 4 migrations, but we have source for 3 migrations up to and including c"),
		},
		{
			desc:        "mismatched state/static",
			name:        "d",
			statenames:  []string{"a", "b", "c"},
			staticnames: []string{"a", "banana", "c"},
			out:         nil,
			err:         errors.New("migration 2 from state (b) does not match name from static list (banana)"),
		},
	}
	for _, tc := range tt {
		state := namesToState(tc.statenames)
		migs := namesToMigs(tc.staticnames)
		out, err := getMigrationsToReverse(tc.name, state, migs)
		assert.Equal(t, migsToNames(out), tc.out)
		assert.Equal(t, err, tc.err)
	}
}
