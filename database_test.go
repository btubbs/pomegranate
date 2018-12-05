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
	url := newURL.String()
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

	goodURL, _ := url.Parse(dburl)
	goodURL.Path = "/goodconnect"
	tt := []struct {
		dbname string
		dburl  string
		err    error
	}{
		{
			dbname: "goodconnect",
			dburl:  goodURL.String(),
			err:    nil,
		},
		{
			dbname: "emptyurl",
			dburl:  "",
			err:    errors.New("empty database url provided"),
		},
		{
			dbname: "badurl",
			dburl:  ":",
			err:    &url.Error{Op: "parse", URL: ":", Err: errors.New("missing protocol scheme")},
		},
	}

	for _, tc := range tt {
		if tc.err == nil {
			master.Exec("CREATE DATABASE " + tc.dbname)
			master.Exec(
				fmt.Sprintf(
					"GRANT ALL PRIVILEGES ON DATABASE %s TO %s",
					tc.dbname, goodURL.User))
			defer master.Exec("DROP DATABASE " + tc.dbname)
		}
		db, err := Connect(tc.dburl)
		assert.Equal(t, tc.err, err)
		if err == nil {
			defer db.Close()
			var result int
			err = db.QueryRow("SELECT 1").Scan(&result)
			assert.Nil(t, err)
			assert.Equal(t, 1, result)
		}
	}
}

func TestGetState(t *testing.T) {
	db, cleanup := freshDB()
	defer cleanup()
	db.Exec(goodMigrations[0].ForwardSQL[0])
	db.Exec(goodMigrations[1].ForwardSQL[0])
	db.Exec(goodMigrations[2].ForwardSQL[0])

	state, err := GetMigrationState(db)
	assert.Nil(t, err)
	names := []string{}
	for _, mr := range state {
		names = append(names, mr.Name)
	}
	expected := []string{
		goodMigrations[0].Name,
		goodMigrations[1].Name,
		goodMigrations[2].Name,
	}
	assert.Equal(t, expected, names)
}

func TestGetLog(t *testing.T) {
	db, cleanup := freshDB()
	defer cleanup()
	db.Exec(goodMigrations[0].ForwardSQL[0])
	db.Exec(goodMigrations[1].ForwardSQL[0])
	db.Exec(goodMigrations[2].ForwardSQL[0])
	db.Exec(goodMigrations[2].BackwardSQL[0])
	db.Exec(goodMigrations[1].BackwardSQL[0])
	db.Exec(goodMigrations[1].ForwardSQL[0])
	db.Exec(goodMigrations[2].ForwardSQL[0])

	log, err := GetMigrationLog(db)

	assert.Nil(t, err)
	names := []string{}
	ops := []string{}
	for _, mr := range log {
		names = append(names, mr.Name)
		ops = append(ops, mr.Op)
	}
	expectedNames := []string{
		goodMigrations[0].Name,
		goodMigrations[1].Name,
		goodMigrations[2].Name,
		goodMigrations[2].Name,
		goodMigrations[1].Name,
		goodMigrations[1].Name,
		goodMigrations[2].Name,
	}
	assert.Equal(t, expectedNames, names)

	expectedOps := []string{
		"INSERT",
		"INSERT",
		"INSERT",
		"DELETE",
		"DELETE",
		"INSERT",
		"INSERT",
	}
	assert.Equal(t, expectedOps, ops)
}

func TestMigrateForwardTo(t *testing.T) {
	db, cleanup := freshDB()
	defer cleanup()
	tt := []struct {
		desc          string
		migrations    []Migration
		migrateToName string
		err           error
		stateName     string
	}{
		{
			desc:          "empty",
			migrations:    []Migration{},
			migrateToName: "foo",
			err:           errors.New("no migrations provided"),
			stateName:     "",
		},
		{
			desc:          "specific",
			migrations:    goodMigrations,
			migrateToName: goodMigrations[2].Name,
			err:           nil,
			stateName:     goodMigrations[2].Name,
		},
		{
			desc:          "all the way",
			migrations:    goodMigrations,
			migrateToName: "",
			err:           nil,
			stateName:     goodMigrations[len(goodMigrations)-1].Name,
		},
	}
	for _, tc := range tt {
		err := MigrateForwardTo(tc.migrateToName, db, tc.migrations, false)
		assert.Equal(t, tc.err, err)
		if tc.err == nil {
			state, _ := GetMigrationState(db)
			assert.Equal(t, tc.stateName, state[len(state)-1].Name)
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
	state, _ := GetMigrationState(db)
	// after migrating back, the latest in state should be the migration
	// right BEFORE the one we just migrated back to (which has been deleted)
	previousName := goodMigrations[0].Name
	assert.Equal(t, previousName, state[len(state)-1].Name)

	// all the way back should fail.
	err = MigrateBackwardTo(goodMigrations[0].Name, db, goodMigrations, false)
	assert.Equal(t,
		errors.New(
			"error running migration: pq: Will not roll back 00001_init.  You must manually drop the migration_state and migration_log tables."),
		err,
	)
}

func TestMigrateFailure(t *testing.T) {
	db, cleanup := freshDB()
	defer cleanup()
	err := MigrateForwardTo("", db, badMigrations, false)
	assert.Equal(t,
		errors.New("error running migration: pq: division by zero"),
		err,
	)
	// the error will have left the DB in a mid-transaction state.  Reset it so we
	// can get state with it.
	_, err = db.Exec("ROLLBACK;")
	assert.Nil(t, err)

	state, err := GetMigrationState(db)
	assert.Nil(t, err)
	// last migration in state should be last good one in badMigrations
	assert.Equal(t, 1, len(state))
	assert.Equal(t,
		badMigrations[0].Name,
		state[len(state)-1].Name,
	)
}

func TestFakeMigrateForwardTo(t *testing.T) {
	db, cleanup := freshDB()
	defer cleanup()
	err := MigrateForwardTo("00001_init", db, goodMigrations, false)
	assert.Nil(t, err)
	err = FakeMigrateForwardTo("", db, goodMigrations, false)
	assert.Nil(t, err)
	state, _ := GetMigrationState(db)
	assert.Equal(t, goodMigrations[len(goodMigrations)-1].Name, state[len(state)-1].Name)
}

func namesToState(names []string) []MigrationRecord {
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
		ForwardSQL: []string{`BEGIN;
CREATE TABLE migration_state (
	name TEXT NOT NULL,
	time TIMESTAMP WITH TIME ZONE DEFAULT now() NOT NULL,
	who TEXT DEFAULT CURRENT_USER NOT NULL,
	PRIMARY KEY (name)
);

CREATE TABLE migration_log (
  id SERIAL PRIMARY KEY,
  time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
  name TEXT NOT NULL,
  op TEXT NOT NULL,
  who TEXT NOT NULL DEFAULT CURRENT_USER
);

CREATE OR REPLACE FUNCTION record_migration() RETURNS trigger AS $$
BEGIN
	IF TG_OP='DELETE' THEN
		INSERT INTO migration_log (name, op) VALUES (
			OLD.name,
			TG_OP
		);
		RETURN OLD;
	ELSE
		INSERT INTO migration_log (name, op) VALUES (
          NEW.name,
          TG_OP
		);
		RETURN NEW;
	END IF;
END;
$$ language plpgsql;

CREATE TRIGGER record_migration AFTER INSERT OR UPDATE OR DELETE ON migration_state
  FOR EACH ROW EXECUTE PROCEDURE record_migration();

INSERT INTO migration_state(name) VALUES ('00001_init');
COMMIT;
`},
		BackwardSQL: []string{`BEGIN;
CREATE OR REPLACE FUNCTION no_rollback() RETURNS void AS $$
BEGIN
  RAISE 'Will not roll back 00001_init.  You must manually drop the migration_state and migration_log tables.';
END;
$$ LANGUAGE plpgsql;

SELECT no_rollback();
COMMIT;
`},
	}, {
		Name: "00002_foobar",
		ForwardSQL: []string{`BEGIN;
CREATE TABLE foo (
  id SERIAL NOT NULL,
  stuff TEXT
);
INSERT INTO migration_state(name) VALUES ('00002_foobar');
COMMIT;
`},
		BackwardSQL: []string{`BEGIN;
DROP TABLE foo;
DELETE FROM migration_state WHERE name='00002_foobar';
COMMIT;
`},
	}, {
		Name: "00003_foobaz",
		ForwardSQL: []string{`BEGIN;
ALTER TABLE foo ADD COLUMN bar TEXT;
INSERT INTO migration_state(name) VALUES ('00003_foobaz');
COMMIT;
`},
		BackwardSQL: []string{`BEGIN;
ALTER TABLE foo DROP COLUMN bar;
DELETE FROM migration_state WHERE name='00003_foobaz';
COMMIT;
`},
	}, {
		Name: "00004_fooquux",
		ForwardSQL: []string{`BEGIN;
CREATE TABLE quux (
  id SERIAL PRIMARY KEY,
  time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
);
INSERT INTO migration_state(name) VALUES ('00004_fooquux');
COMMIT;
`},
		BackwardSQL: []string{`BEGIN;
DROP TABLE quux;
DELETE FROM migration_state WHERE name='00004_fooquux';
COMMIT;
`},
	},
	{
		Name: "00005_seperate",
		ForwardSQL: []string{`
		CREATE TABLE quuxConcurrent  (
		id SERIAL PRIMARY KEY,
		time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now()
		);`, `CREATE INDEX CONCURRENTLY idx_id on quuxConcurrent(id);`,
			`INSERT INTO migration_state(name) VALUES ('00005_seperate');`},
		BackwardSQL: []string{`BEGIN;
		DROP TABLE quuxConcurrent;
		DELETE FROM migration_state WHERE name='00005_seperate';
		COMMIT;
		`},
	},
}

var badMigrations = []Migration{
	{
		Name: "00001_init",
		ForwardSQL: []string{`BEGIN;
CREATE TABLE migration_state (
	name TEXT NOT NULL,
	time TIMESTAMP WITH TIME ZONE DEFAULT now() NOT NULL,
	who TEXT DEFAULT CURRENT_USER NOT NULL,
	PRIMARY KEY (name)
);

CREATE TABLE migration_log (
  id SERIAL PRIMARY KEY,
  time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
  name TEXT NOT NULL,
  op TEXT NOT NULL,
  who TEXT NOT NULL DEFAULT CURRENT_USER
);

CREATE OR REPLACE FUNCTION record_migration() RETURNS trigger AS $$
BEGIN
	IF TG_OP='DELETE' THEN
		INSERT INTO migration_log (name, op) VALUES (
			OLD.name,
			TG_OP
		);
		RETURN OLD;
	ELSE
		INSERT INTO migration_log (name, op) VALUES (
          NEW.name,
          TG_OP
		);
		RETURN NEW;
	END IF;
END;
$$ language plpgsql;

CREATE TRIGGER record_migration AFTER INSERT OR UPDATE OR DELETE ON migration_state
  FOR EACH ROW EXECUTE PROCEDURE record_migration();

INSERT INTO migration_state(name) VALUES ('00001_init');
COMMIT;
`},
		BackwardSQL: []string{`BEGIN;
CREATE OR REPLACE FUNCTION no_rollback() RETURNS void AS $$
BEGIN
  RAISE 'Will not roll back 00001_init.  You must manually drop the migration_state and migration_log tables.';
END;
$$ LANGUAGE plpgsql;

SELECT no_rollback();
COMMIT;
`},
	}, {
		Name: "00002_intentional_fail",
		ForwardSQL: []string{`BEGIN;
SELECT 1 / 0;
INSERT INTO migration_state(name) VALUES ('00002_intentional_fail');
COMMIT;
`},
		BackwardSQL: []string{`BEGIN;
SELECT 1 / 0;
DELETE FROM migration_state WHERE name='00002_intentional_fail';
COMMIT;
`},
	},
}
