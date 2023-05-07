# GoFig
Tool for managing NoSQL DB migrations. This tool is in progress with the aim to support Firestore DB migrations soon.

## Workflow concept:

### New Migration Case
1. Create new migrator with unique name
2. Stage changes
3. Get feedback
4. Push changes
5. Migrator auto stores rollback file

### Load Existing Case (use for rollback)
1. Create migrator with name of existing file. (for example "X_rollback")
2. Load migration
3. Get feedback
4. Push changes
5. Migrator auto stores rollback and logs