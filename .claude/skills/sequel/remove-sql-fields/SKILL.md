---
name: remove-sql-fields
description: TRIGGER when user asks to remove, drop, or delete fields, properties, or columns from a SQL CRUD microservice's object.
---

**CRITICAL**: Read and analyze this microservice before starting. Do NOT explore or analyze other microservices unless explicitly instructed to do so. The instructions in this skill are self-contained to this microservice.

**IMPORTANT**: `MyNoun`, `MyNounKey`, `mynoun`, and `mynounapi` are placeholders for the actual object, its key, directory, and API package of the microservice.

## Workflow

Copy this checklist and track your progress:

```
Removing fields of the object:
- [ ] Step 1: Read Local CLAUDE.md File
- [ ] Step 2: Update the Type Definition of the Object
- [ ] Step 3: Update the Type Definition of the Query
- [ ] Step 4: Update Database Schema
- [ ] Step 5: Remove Mappings of Column Names to Object Fields
- [ ] Step 6: Remove Query Conditions
- [ ] Step 7: Update Integration Tests
- [ ] Step 8: Housekeeping
```

#### Step 1: Read Local `CLAUDE.md` File

Read the local `CLAUDE.md` file in the microservice's directory. It contains microservice-specific instructions that should take precedence over global instructions.

#### Step 2: Update the Type Definition of the Object

Find the type definition of the object in `mynounapi/object.go`.
Remove the appropriate fields from the type definition of the struct.
Remove the field's `dv8` tags and any irrelevant code from the object's `Validate` method.
Clean up any unused imports.

#### Step 3: Update the Type Definition of the Query

Find the type definition of the query in `mynounapi/query.go`.
Remove the appropriate fields from the type definition of the struct.
Remove the field's `dv8` tags and any irrelevant code from the query's `Validate` method.
Clean up any unused imports.

#### Step 4: Update Database Schema

Create a new migration script file in `resources/sql` with an incremental file name. **IMPORTANT**: Do not edit an existing migration file.

Refer to the older `.sql` migration files to identify what if any constraints were associated with the deprecated columns. Use `ALTER TABLE` statements to drop these constraints, if applicable.

```sql
-- DRIVER: mysql
ALTER TABLE my_table DROP CONSTRAINT my_table_constraint_name;

-- DRIVER: pgx
ALTER TABLE my_table DROP CONSTRAINT my_table_constraint_name;

-- DRIVER: mssql
ALTER TABLE my_table DROP CONSTRAINT my_table_constraint_name;
```

Refer to the older `.sql` migration files to identify what if any indices were associated with the deprecated columns. Use `DROP INDEX` statements to drop these indices, if applicable.

```sql
-- DRIVER: mysql
DROP INDEX my_table_idx_deprecated_field ON my_table;

-- DRIVER: pgx
DROP INDEX my_table_idx_deprecated_field;

-- DRIVER: mssql
DROP INDEX my_table_idx_deprecated_field ON my_table;
```

Append `ALTER TABLE` statements to drop the columns.

```sql
-- DRIVER: mysql
ALTER TABLE my_table
	DROP COLUMN deprecated_field,
	DROP COLUMN unused_field;

-- DRIVER: pgx
ALTER TABLE my_table
	DROP COLUMN deprecated_field,
	DROP COLUMN unused_field;

-- DRIVER: mssql
ALTER TABLE my_table DROP COLUMN
	deprecated_field,
	unused_field;

-- DRIVER: sqlite
ALTER TABLE my_table DROP COLUMN deprecated_field;
-- DRIVER: sqlite
ALTER TABLE my_table DROP COLUMN unused_field;
```

**IMPORTANT**: SQLite only supports dropping one column per `ALTER TABLE` statement. Each column must be a separate statement prefixed with `-- DRIVER: sqlite`.

#### Step 5: Remove Mappings of Column Names to Object Fields

Update the mappings of the database column names to their corresponding object fields in `mapColumnsOnInsert`, `mapColumnsOnUpdate` and `mapColumnsOnSelect` in `service.go`. Remove mappings of deprecated columns to deprecated object fields.

#### Step 6: Remove Query Conditions

Remove the query conditions and searchable columns in `prepareWhereClauses` in `service.go` that correspond to the removed fields or columns.

#### Step 7: Update Integration Tests

Remove references to the deprecated fields in `service_test.go`, including in the `NewObject` constructor and `TestMyNoun_ColumnMappings`.

Remove references to the deprecated fields in `mynounapi/object_test.go` and `mynounapi/query_test.go`.

#### Step 8: Housekeeping

Follow the `housekeeping` skill. Removing fields changes only the hand-written domain code (`object.go`, `query.go`, `service.go`, `resources/sql`), not `definition.go`, so the boilerplate regeneration is a no-op; the `Version` bump and `go vet` still apply.
