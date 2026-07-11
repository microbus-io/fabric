---
name: modify-sql-fields
description: TRIGGER when user asks to change, rename, or retype fields, properties, or columns of a SQL CRUD microservice's object.
---

**CRITICAL**: Read and analyze this microservice before starting. Do NOT explore or analyze other microservices unless explicitly instructed to do so. The instructions in this skill are self-contained to this microservice.

**IMPORTANT**: `MyNoun`, `MyNounKey`, `mynoun`, and `mynounapi` are placeholders for the actual object, its key, directory, and API package of the microservice.

## Workflow

Copy this checklist and track your progress:

```
Changing fields of the object:
- [ ] Step 1: Read Local CLAUDE.md File
- [ ] Step 2: Update the Type Definition of the Object
- [ ] Step 3: Update the Type Definition of the Query
- [ ] Step 4: Update Database Schema
- [ ] Step 5: Update Mappings of Column Names to Object Fields
- [ ] Step 6: Update Query Conditions
- [ ] Step 7: Update Integration Tests
- [ ] Step 8: Housekeeping
```

#### Step 1: Read Local `CLAUDE.md` File

Read the local `CLAUDE.md` file in the microservice's directory. It contains microservice-specific instructions that should take precedence over global instructions.

#### Step 2: Update the Type Definition of the Object

Find the type definition of the object in `mynounapi/object.go` in the API directory of the microservice.
Change the fields in the type definition of the struct appropriately.
Change the field's `dv8` tags and the code in the object's `Validate` method appropriately.

#### Step 3: Update the Type Definition of the Query

Find the type definition of the query in `mynounapi/query.go` in the API directory of the microservice.
Change the fields in the type definition of the struct appropriately.
Change the field's `dv8` tags and the code in the query's `Validate` method appropriately.

#### Step 4: Update Database Schema

Skip this step if the changes do not necessitate a database schema change. For example, renaming only a Go field or JSON tag without changing the database column name does not require a schema change.

Create a new migration script file in `resources/sql` with an incremental file name. **IMPORTANT**: Do not edit an existing migration file.

Append `ALTER TABLE` statements to change the type, size or constraints of columns, if applicable. For `pgx`, each aspect of the column (type, nullability, default) requires a separate statement - only include the statements relevant to the change. For `mssql`, if the column already has a default constraint from a prior migration, drop it before adding the new one.

```sql
-- DRIVER: mysql
ALTER TABLE my_table
	MODIFY COLUMN modified_column VARCHAR(384) NOT NULL DEFAULT '',
	MODIFY COLUMN changed_column BIGINT NOT NULL DEFAULT 0;

-- DRIVER: pgx
ALTER TABLE my_table ALTER COLUMN modified_column TYPE VARCHAR(384);
-- DRIVER: pgx
ALTER TABLE my_table ALTER COLUMN modified_column SET NOT NULL;
-- DRIVER: pgx
ALTER TABLE my_table ALTER COLUMN modified_column SET DEFAULT '';
-- DRIVER: pgx
ALTER TABLE my_table ALTER COLUMN changed_column TYPE BIGINT;
-- DRIVER: pgx
ALTER TABLE my_table ALTER COLUMN changed_column SET NOT NULL;
-- DRIVER: pgx
ALTER TABLE my_table ALTER COLUMN changed_column SET DEFAULT 0;

-- DRIVER: mssql
ALTER TABLE my_table DROP CONSTRAINT my_table_df_modified_column;
-- DRIVER: mssql
ALTER TABLE my_table DROP CONSTRAINT my_table_df_changed_column;
-- DRIVER: mssql
ALTER TABLE my_table ALTER COLUMN modified_column NVARCHAR(384) NOT NULL;
-- DRIVER: mssql
ALTER TABLE my_table ALTER COLUMN changed_column BIGINT NOT NULL;
-- DRIVER: mssql
ALTER TABLE my_table ADD CONSTRAINT my_table_df_modified_column DEFAULT '' FOR modified_column;
-- DRIVER: mssql
ALTER TABLE my_table ADD CONSTRAINT my_table_df_changed_column DEFAULT 0 FOR changed_column;
```

Append `ALTER TABLE` or `EXEC` statements to rename columns, if applicable. For `mysql`, prefer using the new `RENAME COLUMN` syntax over the old `CHANGE COLUMN` syntax.

```sql
-- DRIVER: mysql
ALTER TABLE my_table
    RENAME COLUMN old_column TO new_column;

-- DRIVER: pgx
ALTER TABLE my_table
    RENAME COLUMN old_column TO new_column;

-- DRIVER: mssql
EXEC sp_rename 'my_table.old_column', 'new_column', 'COLUMN';
```

Refer to the older `.sql` migration files to identify what if any indices were associated with renamed columns. Append statements to the migration script to rename these indices, if applicable.

```sql
-- DRIVER: mysql
ALTER TABLE my_table RENAME INDEX my_table_idx_old_column TO my_table_idx_new_column;

-- DRIVER: pgx
ALTER INDEX my_table_idx_old_column RENAME TO my_table_idx_new_column;

-- DRIVER: mssql
EXEC sp_rename 'my_table_idx_old_column', 'my_table_idx_new_column', 'INDEX';
```

#### Step 5: Update Mappings of Column Names to Object Fields

Update the mappings of the modified database column names to their corresponding object fields in `mapColumnsOnInsert`, `mapColumnsOnUpdate` and `mapColumnsOnSelect` in `service.go`.

#### Step 6: Update Query Conditions

Adjust the query conditions and searchable columns in `prepareWhereClauses` in `service.go` that correspond to the changed fields or columns.

#### Step 7: Update Integration Tests

Adjust references to the changed fields in `service_test.go`, including in the `NewObject` constructor and `TestMyNoun_ColumnMappings`.

Adjust references to the changed fields in `mynounapi/object_test.go` and `mynounapi/query_test.go`.

#### Step 8: Housekeeping

Follow the `housekeeping` skill. Modifying fields changes only the hand-written domain code (`object.go`, `query.go`, `service.go`, `resources/sql`), not `definition.go`, so the boilerplate regeneration is a no-op; the `Version` bump and `go vet` still apply.
