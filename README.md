# GoFig
GoFig is a tool for managing NoSQL DB migrations. At this time, GoFig supports Firestore DB migrations. The GoFig migration manager makes a few guarantees:
1. The end user will have visibility into all migration changes and effects prior to pushing a migration. The user can then decide to push or cancel the migration.
2. When a migration is pushed to the database, all the changes presented to the end user are implemented and no other changes/effects are implemented by the migrator.
3. All changes made by the migrator in a migration job can be completely reversed by staging and pushing the generated `_rollback` migration file.

## Initialize Migrator
The migrator is initialized with a few configuration parameters.
- `KeyPath` contains the path location of your firestore admin key.
- `StoragePath` contains a path to any local folder the migrator can use to save or load migration files.
- `Name` is a unique identifier for the migration which will be used when loading or creating migration files.
```go
import fig "github.com/aaronhough/GoFig"

config := fig.Config{
    KeyPath: "~/project/.keys/my-admin-key.json",
    StoragePath: "~/project/storage",
    Name: "my-migration",
}

fg, err := fig.New(config)

defer fg.Close()
```

## Stage a Migration
Once the migrator is initialized, the contents of the migration must be staged. The migration can be staged one of two ways.
1. Stage each change into the migrator during runtime using the `stage()` utility.
    - Note available actions are `Add, Update, Set, Delete`
    - To save a staged migration to storage for later use, leverage the `SaveToFile()` utility
2. If the `Name` of the migration corresponds to an existing migration file, use the `LoadFromFile()` utility to load all changes.
```go
data := map[string]any{
    "foo": "bar",
    "fiz": false,
    "buz": map[string]any{
        "a": []any{ 2, 2, 3 },
    },
}

fg.Stage().Add("fig/fog", data)
fg.Stage().Update("fig/bog", map[string]any{ "hello": "world" })
```

## Run a Migration
Use the `ManageStagedMigration()` utility to initiate a responsive CLI migration process. 
- The process will present the staged changes and allow you to push or cancel. 
- If a migration is pushed to the database, a new migration and corresponding `_rollback` file is created in the 'StoragePath' folder.
```go
// Launches interactive shell
fg.ManageStagedMigration()
```

## Rollback
Rollbacks are identical to any other migration job in format and protocol. To rollback a migration that has been pushed:
- Initialize a new migration with the `Name` of the original migration appended by "_rollback" (For example "my-migration_rollback").
- When staging the migration, use the `LoadFromFile()` method outlined above under the staging section.
- Run the migration as you would any other job. The CLI process will be the same.
- You will recieve a rollback to your rollback if the job is pushed.
```go
config := fig.Config{
    KeyPath: "~/project/.keys/my-admin-key.json",
    StoragePath: "~/project/storage",
    Name: "my-migration_rollback",
}

fg, err := fig.New(config)
defer fg.Close()

fg.LoadFromFile()
fg.ManageStagedMigration()
```

## Complex types
To ensure proper handling of complex types, follow these prococols.
- For times, feel free to use the standard time utilities.
- To mark a key or nested key for deletion, use the GoFig `DeleteField` utility.
- To use document references as values, use the GoFig `RefField` utility.
```go
data := map[string]any{
    "ref": fg.RefField("fig/fog"),
    "time": time.Now(),
    "prev": fg.DeleteField(),
}
```

## To Do
- Allow for specific type declarations in nested lists/maps as opposed to only `[]any` and `map[string]any`