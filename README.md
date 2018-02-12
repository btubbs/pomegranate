# Pomegranate [![Build Status](https://travis-ci.org/btubbs/pomegranate.svg?branch=master)](https://travis-ci.org/btubbs/pomegranate)

Pomegranate is a tool for creating and running schema migrations for the
Postgres database, emphasizing safety and transparency.  All migrations are
run in transactions that will be automatically rolled back if an exception is
raised, ensuring the database is not left in a half-migrated state. Further, the
.sql migration files created by Pomegranate are the exact SQL that will be run
at migration time; you will get the same result whether you call `pmg forward`,
or feed a forward.sql file to `psql`, or attach the forward.sql file to a change
control ticket and have your DBA run it. Postgres' best-in-class support for
[transactional
DDL](https://wiki.postgresql.org/wiki/Transactional_DDL_in_PostgreSQL:_A_Competitive_Analysis)
makes this safety and transparency possible.



## Installation

For now, pomegranate has to be built from source:

    $ git clone git@github.com:btubbs/pomegranate.git
    $ cd pomegranate
    $ go install cmd/pmg.go

## Usage

You can use Pomegranate in two ways:

1. As a standalone command line tool (`pmg`).
2. As a library (`package pomegranate`) in your own Go project.

### Using `pmg`
#### Create initial migration

Use the `pmg init` command to create your first migration, which will be
responsible for creating the `migrations_history` table.  It will create the
migration in the current directory, or you can specify a different one with the
`--dir` option.


    $ pmg init
    Migration stubs written to 00001_init

The `00001_init` directory should now exist, and contain `forward.sql` and
`backward.sql` files.  You don't need to edit these initial migrations.

#### Create more migrations

Migrations containing your own custom changes should be made with the `pmg new`
command.

    $ pmg new add_customers_table
    Migration stubs written to 00002_add_customers_table

Note that the `00002` prefix has been prepended to the name you provided.
Migrations are run in the order they appear in the file system.  The auto
numbering ensures that this ordering is consistent.

As with `init`, the `new` command creates `forward.sql` and `backward.sql`
files.  Unlike `init`, these are just stubs.  You will need to edit these files
and add your own commands (e.g. `CREATE TABLE...`).  The stub files try to make
it obvious where your commands should go:

    $ cat 00002_add_customers_table/forward.sql 
    BEGIN;
    -- vvvvvvvv PUT FORWARD MIGRATION CODE BELOW HERE vvvvvvvv

    SELECT 1 / 0; -- delete this line

    -- ^^^^^^^^ PUT FORWARD MIGRATION CODE ABOVE HERE ^^^^^^^^
    INSERT INTO migration_history(name) VALUES ('00002_add_customers_table');
    COMMIT;

The `SELECT 1 / 0;` line in the stub is a safeguard against accidentally running
an empty migration.  You should replace it with your own commands.

Be sure to also add the necessary commands to `backward.sql` to safely roll back
the changes in `forward.sql`, in case you decide they were a bad idea.

#### Run migrations

Use the `forward` command to run all migrations not yet recorded in the
`migration_history` table.  Pomegranate will connect to the database specified
in the `DATABASE_URL` environment variable, or you can supply a database URL
with the `--dburl` option.

    $ pmg forward
    Connecting to database 'readme' on host ''
    Forward migrations that will be run:
    00001_init
    00002_add_customers_table
    Run these migrations? (y/n) y
    Running 00001_init... Success!
    Running 00002_add_customers_table... Success!
    Done

If you don't want to run all the migrations, you can use the `forwardto` command
instead:

    $ pmg forwardto 00001_init
    Connecting to database 'readme' on host ''
    Forward migrations that will be run:
    00001_init
    Run these migrations? (y/n) y
    Running 00001_init... Success!
    Done

#### Roll back migrations

Rolling back is done with the `backwardto` command.  This will run the
`backward.sql` file for all migrations that have already been run, up to and
including the one specified in the command.

    $ pmg backwardto 00002_add_customers_table
    Connecting to database 'readme' on host ''
    Backward migrations that will be run:
    00002_add_customers_table
    Run these migrations? (y/n) y
    Running 00002_add_customers_table... Success!
    Done

Unlike going forward, `pmg` does NOT provide a `backward` command that will
migrate all the way back.  You must always provide a specific migration name to
`backwardto`.

#### View migration history

The `history` command will show all migrations recorded in the
`migration_history` table:

    $ pmg history
    Connecting to database 'readme' on host ''
    NAME       | WHEN                                 | WHO
    00001_init | 2018-02-11 20:48:51.827197 -0700 MST | postgres

### Using the pomegranate package
#### Ingest migrations
