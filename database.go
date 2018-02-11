package pomegranate

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"

	_ "github.com/lib/pq"
)

// GetMigrationHistory returns the list of migration records stored in the
// database's migation_history table.  If that table does not exist, it returns
// an empty list.
func GetMigrationHistory(db *sql.DB) ([]MigrationRecord, error) {
	// first see if the migration_history table exists
	var exists bool
	err := db.QueryRow(`
      SELECT EXISTS (
         SELECT 1 
         FROM   pg_tables
         WHERE  schemaname = 'public'
         AND    tablename = 'migration_history'
       );`).Scan(&exists)
	if err != nil {
		return nil, err
	}

	if !exists {
		return []MigrationRecord{}, nil
	}
	rows, err := db.Query("SELECT name, time, who FROM migration_history ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("get past migrations: %v", err)
	}
	defer rows.Close()
	pastMigrations := []MigrationRecord{}
	for rows.Next() {
		var pm MigrationRecord
		if err := rows.Scan(&pm.Name, &pm.Time, &pm.Who); err != nil {
			return nil, fmt.Errorf("get past migrations: %v", err)
		}
		pastMigrations = append(pastMigrations, pm)
	}
	return pastMigrations, nil
}

func runMigrationSQL(db *sql.DB, name, sqlToRun string) error {
	fmt.Printf("Running %s... ", name)
	_, err := db.Exec(sqlToRun)
	if err != nil {
		return fmt.Errorf("error: %v", err)
	}
	fmt.Println("Success!")
	return nil
}

func MigrateBackwardTo(name string, db *sql.DB, allMigrations []Migration, confirm bool) error {
	if len(allMigrations) == 0 {
		return errors.New("no migrations provided")
	}
	history, err := GetMigrationHistory(db)
	if err != nil {
		return fmt.Errorf("could not get migration history: %v", err)
	}
	// if nothing in history, nothing to do. error
	if len(history) == 0 {
		return errors.New("history is empty. cannot migrate back")
	}
	toRun, err := getMigrationsToReverse(name, history, allMigrations)
	if err != nil {
		return err
	}
	// get confirmation on the list of backward migrations we're going to run
	if confirm {
		if err := getConfirm(toRun, "Backward"); err != nil {
			return err
		}
	}
	// run the migrations
	for _, mig := range toRun {
		err = runMigrationSQL(db, mig.Name, mig.BackwardSQL)
		if err != nil {
			return err
		}
	}
	return nil
}

// MigrateForwardTo will run all forward migrations that have not yet been run,
// up to and including the one specified by `name`.
// To run all un-run migrations, set `name` to an empty string.
func MigrateForwardTo(name string, db *sql.DB, allMigrations []Migration, confirm bool) error {
	if len(allMigrations) == 0 {
		return errors.New("no migrations provided")
	}
	history, err := GetMigrationHistory(db)
	if err != nil {
		return fmt.Errorf("could not get migration history: %v", err)
	}
	if nameInHistory(name, history) {
		return fmt.Errorf("migration '%s' has already been run", name)
	}
	if name == "" {
		name = allMigrations[len(allMigrations)-1].Name
	}
	forwardMigrations, err := getForwardMigrations(history, allMigrations)
	if len(forwardMigrations) == 0 {
		fmt.Println("No migrations to run")
		return nil
	}
	if !nameInMigrationList(name, forwardMigrations) {
		return fmt.Errorf("migration '%s' not in list of un-run migrations")
	}
	// trim forwardMigrations later than name
	toRun, err := trimMigrationsTail(name, forwardMigrations)
	if err != nil {
		return err
	}
	if confirm {
		if err := getConfirm(toRun, "Forward"); err != nil {
			return err
		}
	}
	// run migrations
	for _, mig := range toRun {
		err = runMigrationSQL(db, mig.Name, mig.ForwardSQL)
		if err != nil {
			return err
		}
	}
	return nil
}

// Connect calls sql.Open for you, specifying the Postgres driver and printing
// the DB name and host to stdout so you can check that you're connecting to the
// right place before continuing.
func Connect(dburl string) (*sql.DB, error) {
	// Failure to set the DATABASE_URL env var or provide the dburl command line
	// flag could result in an empty dburl here.  Catch that.
	if dburl == "" {
		return nil, errors.New("empty database url provided")
	}
	url, err := url.Parse(dburl)
	if err != nil {
		return nil, err
	}
	// trim leading slash
	dbname := strings.Trim(url.Path, "/")
	fmt.Printf("Connecting to database '%s' on host '%s'\n", dbname, url.Host)
	return sql.Open("postgres", dburl)
}
