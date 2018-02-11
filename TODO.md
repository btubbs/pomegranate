OK SO I got a simple run working.  What's next?
- migrateforwardto should just take a db, not a history. DONE
- pmg forward should default to latest migration if you don't supply a name DONE
- get it reading the DB from the environment. DONE
- add forward cmd DONE
- and backward command DONE
- add tests
  - init stubs DONE
  - regular stubs (including auto numbering) DONE
  - ingest DONE
  - history DONE
  - running init migration (forward and back) DONE
  - running forward all DONE
  - running forward to specific DONE
  - running back to specific DONE
- add README

BACKLOG
- support passing in a schema other than "public"
  So, as far as DDL goes, that's not really necessary.  That could all be done
  in the SQL itself.

  The only exception is the migration_history table.  If you wanted that to be
  something else, You'd have to change it in:
  - migration templates
  - gethistory

OK SO I've tested all the happy paths.  Now we need to test error cases.

The coverage tool is really pretty badass.  Really.
