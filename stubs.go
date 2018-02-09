package pomegranate

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

const leadingDigits = 5
const forwardFile = "forward.sql"
const backwardFile = "backward.sql"

const InitForwardTmpl = `BEGIN;
CREATE TABLE migration_history (
	name TEXT NOT NULL,
	time TIMESTAMP WITH TIME ZONE DEFAULT now() NOT NULL,
	who TEXT DEFAULT CURRENT_USER NOT NULL,
	PRIMARY KEY (name)
);

INSERT INTO migration_history(name) VALUES ('%s');
COMMIT;
`

const InitBackwardTmpl = `BEGIN;
CREATE OR REPLACE FUNCTION safe_drop_history() RETURNS void AS $$
BEGIN
	IF (SELECT count(*) FROM migration_history)>0 THEN
		DROP TABLE migration_history;
	ELSE
		RAISE 'migration_history table not empty';
	END IF;
END;
$$ LANGUAGE plpgsql;

DELETE FROM migration_history WHERE name='%s';
SELECT safe_drop_history();
DROP FUNCTION safe_drop_history();
COMMIT;
`

const ForwardTmpl = `BEGIN;
SELECT 1 / 0; -- Delete this line and replace with your own code that migrates forward.
INSERT INTO migration_history(name) VALUES ('%s');
COMMIT;
`

const BackwardTmpl = `BEGIN;
SELECT 1 / 0; -- Delete this line and replace with your own code that migrates backward.
DELETE FROM migration_history WHERE name='%s';
COMMIT;
`

func zeroPad(num, digits int) string {
	return fmt.Sprintf("%"+fmt.Sprintf("0%dd", digits), num)
}

func isMigration(dir string) bool {
	pat := fmt.Sprintf(`^[\d]{%d}_.*$`, leadingDigits)
	match, err := regexp.MatchString(pat, dir)
	if err != nil {
		return false
	}
	return match
}

// return a sorted list of subdirs that match our pattern
func getMigrationFileNames(dir string) ([]string, error) {
	names := []string{}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("error listing migration files: %v", err)
	}

	for _, file := range files {
		name := file.Name()
		if err != nil {
			return nil, err
		}
		if file.IsDir() && isMigration(name) {
			names = append(names, name)
		}
	}
	return names, nil
}

func getLatestMigrationNumber(dir string) (int, error) {
	files, err := getMigrationFileNames(dir)
	if err != nil {
		return 0, fmt.Errorf("error getting migration number: %v", err)
	}
	if len(files) == 0 {
		return 0, nil
	}
	last := files[len(files)-1]
	parts := strings.Split(last, "_")
	num, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("error getting migration number: %v", err)
	}
	return num, nil
}

func writeStubs(dir, name, forwardSQL, backwardSQL string) error {
	newFolder := path.Join(dir, name)
	err := os.Mkdir(newFolder, 0755)
	if err != nil {
		return fmt.Errorf("error creating migration directory %s: %v", newFolder, err)
	}

	err = ioutil.WriteFile(path.Join(newFolder, "forward.sql"), []byte(forwardSQL), 0644)
	if err != nil {
		return fmt.Errorf("error writing migration file: %v", err)
	}
	err = ioutil.WriteFile(path.Join(newFolder, "backward.sql"), []byte(backwardSQL), 0644)
	if err != nil {
		return fmt.Errorf("error writing migration file: %v", err)
	}
	fmt.Printf("Migration stubs written to %s\n", newFolder)
	return nil
}

func makeStubName(numPart int, namePart string) string {
	return fmt.Sprintf("%s_%s", zeroPad(numPart, leadingDigits), namePart)
}

func NewMigration(dir, name string) error {
	latestNum, err := getLatestMigrationNumber(dir)
	if err != nil {
		return fmt.Errorf("error making new migration: %v", err)
	}
	newName := makeStubName(latestNum+1, name)
	forwardSQL := fmt.Sprintf(ForwardTmpl, newName)
	backwardSQL := fmt.Sprintf(BackwardTmpl, newName)
	err = writeStubs(dir, newName, forwardSQL, backwardSQL)
	if err != nil {
		return fmt.Errorf("error making new migration: %v", err)
	}
	return nil
}

func InitMigration(dir string) error {
	name := makeStubName(1, "init")
	forwardSQL := fmt.Sprintf(InitForwardTmpl, name)
	backwardSQL := fmt.Sprintf(InitBackwardTmpl, name)
	err := writeStubs(dir, name, forwardSQL, backwardSQL)
	if err != nil {
		return err
	}
	return nil
}
