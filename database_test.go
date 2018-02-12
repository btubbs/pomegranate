package pomegranate

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var dburl string
var master *sql.DB
var r *rand.Rand

func randName() string {
	b := make([]byte, 8)
	chars := "abcdefghijklmnopqrstuvwxyz"
	for i := range b {
		b[i] = chars[r.Intn(len(chars))]
	}
	return string(b)
}

// freshDB returns a connection to a new, empty, randomly named DB, and a
// function that will close it and delete the random DB when called
func freshDB() (*sql.DB, func()) {
	name := "pmgtest" + randName()
	master.Exec("CREATE DATABASE " + name)

	newURL, _ := url.Parse(dburl)
	newURL.Path = "/" + name
	url := fmt.Sprintf(newURL.String())
	db, _ := sql.Open("postgres", url)
	cleanup := func() {
		db.Close()
		master.Exec("DROP DATABASE " + name)
	}
	return db, cleanup
}

func TestMain(m *testing.M) {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
	var err error
	dburl = os.Getenv("DATABASE_URL")
	if dburl == "" {
		dburl = "postgres://postgres@/postgres?sslmode=disable"
	}
	master, err = sql.Open("postgres", dburl)
	if err != nil {
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestConnect(t *testing.T) {
	tt := []struct {
		dbname string
		dburl  string
		err    error
		urlerr *url.Error
	}{
		{
			dbname: "goodconnect",
			dburl:  "postgres://postgres@/goodconnect?sslmode=disable",
			err:    nil,
			urlerr: nil,
		},
		{
			dbname: "emptyurl",
			dburl:  "",
			err:    errors.New("empty database url provided"),
			urlerr: nil,
		},
		{
			dbname: "badurl",
			dburl:  ":",
			err:    nil,
			urlerr: &url.Error{"parse", ":", errors.New("missing protocol scheme")},
		},
	}

	for _, tc := range tt {
		master.Exec("CREATE DATABASE " + tc.dbname)
		defer master.Exec("DROP DATABASE " + tc.dbname)
		db, err := Connect(tc.dburl)
		if tc.urlerr != nil {
			assert.Equal(t, tc.urlerr, err)
		} else {
			assert.Equal(t, tc.err, err)
		}
		if err == nil {
			defer db.Close()
			var result int
			err = db.QueryRow("SELECT 1").Scan(&result)
			assert.Nil(t, err)
			assert.Equal(t, 1, result)
		}
	}
}

func TestGetHistory(t *testing.T) {
	db, cleanup := freshDB()
	defer cleanup()
	db.Exec(goodMigrations[0].ForwardSQL)
	db.Exec(goodMigrations[1].ForwardSQL)
	db.Exec(goodMigrations[2].ForwardSQL)

	history, err := GetMigrationHistory(db)
	assert.Nil(t, err)
	names := []string{}
	for _, mr := range history {
		names = append(names, mr.Name)
	}
	expected := []string{
		goodMigrations[0].Name,
		goodMigrations[1].Name,
		goodMigrations[2].Name,
	}
	assert.Equal(t, expected, names)
}

func TestMigrateForwardTo(t *testing.T) {
	db, cleanup := freshDB()
	defer cleanup()
	tt := []struct {
		desc          string
		migrations    []Migration
		migrateToName string
		err           error
		historyName   string
	}{
		{
			desc:          "empty",
			migrations:    []Migration{},
			migrateToName: "foo",
			err:           errors.New("no migrations provided"),
			historyName:   "",
		},
		{
			desc:          "specific",
			migrations:    goodMigrations,
			migrateToName: goodMigrations[2].Name,
			err:           nil,
			historyName:   goodMigrations[2].Name,
		},
		{
			desc:          "all the way",
			migrations:    goodMigrations,
			migrateToName: "",
			err:           nil,
			historyName:   goodMigrations[len(goodMigrations)-1].Name,
		},
	}
	for _, tc := range tt {
		err := MigrateForwardTo(tc.migrateToName, db, tc.migrations, false)
		assert.Equal(t, tc.err, err)
		if tc.err == nil {
			history, _ := GetMigrationHistory(db)
			assert.Equal(t, tc.historyName, history[len(history)-1].Name)
		}
	}
}

func TestMigrateBackwardTo(t *testing.T) {
	db, cleanup := freshDB()
	defer cleanup()
	MigrateForwardTo("", db, goodMigrations, false)
	name := goodMigrations[1].Name
	err := MigrateBackwardTo(name, db, goodMigrations, false)
	assert.Nil(t, err)
	history, _ := GetMigrationHistory(db)
	// after migrating back, the latest in history should be the migration
	// right BEFORE the one we just migrated back to (which has been deleted)
	previousName := goodMigrations[0].Name
	assert.Equal(t, previousName, history[len(history)-1].Name)

	// all the way back
	err = MigrateBackwardTo(goodMigrations[0].Name, db, goodMigrations, false)
	assert.Nil(t, err)
	history, _ = GetMigrationHistory(db)
	assert.Equal(t, []MigrationRecord{}, history)
}

func TestMigrateFailure(t *testing.T) {
	db, cleanup := freshDB()
	defer cleanup()
	err := MigrateForwardTo("", db, badMigrations, false)
	assert.Equal(t,
		errors.New("error: pq: division by zero"),
		err,
	)
	// the error will have left the DB in a mid-transaction state.  Reset it so we
	// can get history with it.
	_, err = db.Exec("ROLLBACK;")
	assert.Nil(t, err)

	history, err := GetMigrationHistory(db)
	assert.Nil(t, err)
	// last migration in history should be last good one in badMigrations
	assert.Equal(t, 1, len(history))
	assert.Equal(t,
		badMigrations[0].Name,
		history[len(history)-1].Name,
	)
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

var goodMigrations = []Migration{
	{
		Name: "00001_init",
		ForwardSQL: `BEGIN;
CREATE TABLE migration_history (
	name TEXT NOT NULL,
	time TIMESTAMP WITH TIME ZONE DEFAULT now() NOT NULL,
	who TEXT DEFAULT CURRENT_USER NOT NULL,
	PRIMARY KEY (name)
);

INSERT INTO migration_history(name) VALUES ('00001_init');
COMMIT;
`,
		BackwardSQL: `BEGIN;
CREATE OR REPLACE FUNCTION safe_drop_history() RETURNS void AS $$
BEGIN
	IF (SELECT count(*) FROM migration_history)=0 THEN
		DROP TABLE migration_history;
	ELSE
		RAISE 'migration_history table not empty';
	END IF;
END;
$$ LANGUAGE plpgsql;

DELETE FROM migration_history WHERE name='00001_init';
SELECT safe_drop_history();
DROP FUNCTION safe_drop_history();
COMMIT;
`,
	}, {
		Name: "00002_foobar",
		ForwardSQL: `BEGIN;
CREATE TABLE foo (
  id SERIAL NOT NULL,
  stuff TEXT
);
INSERT INTO migration_history(name) VALUES ('00002_foobar');
COMMIT;
`,
		BackwardSQL: `BEGIN;
DROP TABLE foo;
DELETE FROM migration_history WHERE name='00002_foobar';
COMMIT;
`,
	}, {
		Name: "00003_foobaz",
		ForwardSQL: `BEGIN;
ALTER TABLE foo ADD COLUMN bar TEXT;
INSERT INTO migration_history(name) VALUES ('00003_foobaz');
COMMIT;
`,
		BackwardSQL: `BEGIN;
ALTER TABLE foo DROP COLUMN bar;
DELETE FROM migration_history WHERE name='00003_foobaz';
COMMIT;
`,
	}, {
		Name: "00004_fooquux",
		ForwardSQL: `BEGIN;
CREATE TABLE quux (
  id SERIAL PRIMARY KEY,
  time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);
INSERT INTO migration_history(name) VALUES ('00004_fooquux');
COMMIT;
`,
		BackwardSQL: `BEGIN;
DROP TABLE quux;
DELETE FROM migration_history WHERE name='00004_fooquux';
COMMIT;
`,
	},
}

var badMigrations = []Migration{
	{
		Name: "00001_init",
		ForwardSQL: `BEGIN;
CREATE TABLE migration_history (
	name TEXT NOT NULL,
	time TIMESTAMP WITH TIME ZONE DEFAULT now() NOT NULL,
	who TEXT DEFAULT CURRENT_USER NOT NULL,
	PRIMARY KEY (name)
);

INSERT INTO migration_history(name) VALUES ('00001_init');
COMMIT;
`,
		BackwardSQL: `BEGIN;
CREATE OR REPLACE FUNCTION safe_drop_history() RETURNS void AS $$
BEGIN
	IF (SELECT count(*) FROM migration_history)=0 THEN
		DROP TABLE migration_history;
	ELSE
		RAISE 'migration_history table not empty';
	END IF;
END;
$$ LANGUAGE plpgsql;

DELETE FROM migration_history WHERE name='00001_init';
SELECT safe_drop_history();
DROP FUNCTION safe_drop_history();
COMMIT;
`,
	}, {
		Name: "00002_intentional_fail",
		ForwardSQL: `BEGIN;
SELECT 1 / 0;
INSERT INTO migration_history(name) VALUES ('00002_intentional_fail');
COMMIT;
`,
		BackwardSQL: `BEGIN;
SELECT 1 / 0;
DELETE FROM migration_history WHERE name='00002_intentional_fail';
COMMIT;
`,
	},
}
