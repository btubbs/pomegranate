OK SO I got a simple run working.  What's next?
- migrateforwardto should just take a db, not a history. DONE
- pmg forward should default to latest migration if you don't supply a name DONE
- get it reading the DB from the environment. DONE
- add forward cmd DONE
- and backward command
- add README
- add tests
  - init stubs
  - regular stubs (including auto numbering)
  - ingest
  - history
  - running init migration (forward and back)
  - running forward all
  - running forward to specific
  - running back to specific

BACKLOG
- support passing in a schema other than "public"
