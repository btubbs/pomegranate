package pomegranate

import (
	"database/sql"
	"fmt"
)

func RunMigrationForward(db *sql.DB, m Migration) error {
	_, err := db.Exec(m.ForwardSQL)
	if err != nil {
		return fmt.Errorf("run migration forward %s: %v", m.Name, err)
	}
	return nil
}

func GetMigrationHistory(db *sql.DB) ([]MigrationRecord, error) {
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

func MigrateForwardTo(dir, name string) error {
	// get a db connection
	// see what migration was run most recently there
	// figure out our start/end
	return nil
}
