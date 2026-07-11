---
name: add-sql-fields
description: TRIGGER when user asks to add fields, properties, or columns to a SQL CRUD microservice's object.
---

**CRITICAL**: Read and analyze this microservice before starting. Do NOT explore or analyze other microservices unless explicitly instructed to do so. The instructions in this skill are self-contained to this microservice.

**IMPORTANT**: `MyNoun`, `MyNounKey`, `mynoun`, and `mynounapi` are placeholders for the actual object, its key, directory, and API package of the microservice.

**IMPORTANT**: Do not remove the `Example` field or code related to it from the code since it is required by various tests.

## Workflow

Copy this checklist and track your progress:

```
Adding fields to the object:
- [ ] Step 1: Read Local CLAUDE.md File
- [ ] Step 2: Update the Type Definition of the Object
- [ ] Step 3: Update the Type Definition of the Query
- [ ] Step 4: Update Database Schema
- [ ] Step 5: Map Column Names to Object Fields
- [ ] Step 6: Add Query Conditions
- [ ] Step 7: Update Integration Tests
- [ ] Step 8: Housekeeping
```

#### Step 1: Read Local `CLAUDE.md` File

Read the local `CLAUDE.md` file in the microservice's directory. It contains microservice-specific instructions that should take precedence over global instructions.

#### Step 2: Update the Type Definition of the Object

Find the type definition of the object in `mynounapi/object.go`.
Add the new fields to the type definition of the struct.
Fields can be primitive or complex types that serialize to JSON.
Be sure to include a JSON tag. Use camelCase for the JSON name, and specify to `omitzero`.

When referring to an object persisted by another microservices, use its respective key.

Express field-level constraints (trimming, length limits, value ranges, required values) as `dv8` tags on the
fields; they are enforced automatically when the object is validated (see the `add-type` skill).

```go
type MyNoun struct {
	Key MyNounKey `json:"key,omitzero"`

	// HINT: Define the fields of the object here, with dv8 tags for field-level constraints
	MyFieldString   string             `json:"myFieldString,omitzero" dv8:"trim,len<=256"`
	MyFieldInteger  int                `json:"myFieldInteger,omitzero" dv8:"val>=0"`
	MyFieldNullable string             `json:"myFieldNullable,omitzero" dv8:"trim,len<=256"`
	MyFieldTime     time.Time          `json:"myFieldTime,omitzero"`
	MyFieldTags     map[string]string  `json:"myFieldTags,omitzero" dv8:"each len<=256"`
	
	MyOtherObjectKey otherobjectapi.OtherObjectKey `json:"myOtherObjectKey,omitzero" dv8:"notzero"`
}
```

Constraints that the `dv8` tag grammar cannot express - comparisons against the current time, invariants that span
multiple fields - go in the object's `Validate` method, which is called automatically after the tags are enforced.

```go
// Validate validates invariants that span multiple fields of the object.
// It is called automatically when the object is validated, after the dv8 field tags are enforced.
func (obj *MyNoun) Validate(ctx context.Context) error {
	// ...

	// HINT: Validate invariants that span multiple fields here as required
	if obj.MyFieldTime.After(time.Now()) {
		return errors.New("MyFieldTime must not be in the future")
	}
	return nil
}
```

#### Step 3: Update the Type Definition of the Query

Find the type definition of the `Query` in `mynounapi/query.go`.
To allow filtering by the new fields, add them to the type definition of the struct.
Fields can be primitive or complex types that serialize to JSON.
Be sure to include a JSON tag. Use camelCase for the JSON name, and specify to `omitzero`.

When referring to a parent object that is persisted by another SQL CRUD microservices, use its respective key as the field type.

Express field-level constraints as `dv8` tags on the fields, matching the constraints of the object fields they
filter by (e.g. the same length limit).

```go
type Query struct {
	Key MyNounKey `json:"key,omitzero"`

	// HINT: Define the fields of the object here, with dv8 tags for field-level constraints
	MyFieldInteger  int       `json:"myFieldInteger,omitzero" dv8:"val>=0"`
	MyFieldNullable string    `json:"myFieldNullable,omitzero" dv8:"trim,len<=256"`
	MyFieldTimeGTE  time.Time `json:"myFieldTimeStart,omitzero"`
	MyFieldTimeLT   time.Time `json:"myFieldTimeEnd,omitzero"`
	
	ParentKey parentapi.ParentKey `json:"parentKey,omitzero" dv8:"notzero"`
}
```

Constraints that the `dv8` tag grammar cannot express go in the `Query`'s `Validate` method, which is called
automatically after the tags are enforced.

```go
// Validate validates the filtering options of the query that cannot be expressed as dv8 field tags.
// It is called automatically when the query is validated, after the dv8 field tags are enforced.
func (q *Query) Validate(ctx context.Context) error {
	// ...

	// HINT: Validate filtering options that cannot be expressed as dv8 tags here as required
	if q.MyFieldTimeGTE.After(time.Now()) {
		return errors.New("MyFieldTimeGTE must not be in the future")
	}
	if q.MyFieldTimeLT.After(time.Now()) {
		return errors.New("MyFieldTimeLT must not be in the future")
	}
	if q.MyFieldTimeGTE.After(q.MyFieldTimeLT) {
		return errors.New("MyFieldTimeGTE must not be after MyFieldTimeLT")
	}
	return nil
}
```

#### Step 4: Update Database Schema

Create a new migration script file in `resources/sql` with an incremental file name. **IMPORTANT**: Do not edit an existing migration file.

Append `ALTER TABLE` statements to define the schema of the new columns. Define a `DEFAULT` value for all new columns that are `NOT NULL` in order to avoid the migration from failing on tables already populated with data. Columns holding IDs of parent objects should be named after the table they refer to with an `_id` suffix, e.g. `parent_table_id`.

```sql
-- DRIVER: mysql
ALTER TABLE my_noun
	ADD my_field_integer BIGINT NOT NULL DEFAULT 0,
	ADD my_field_nullable TEXT NULL,
	ADD my_field_time DATETIME NULL,
	ADD my_field_tags MEDIUMBLOB NULL,
	ADD parent_table_id BIGINT NOT NULL DEFAULT 0;

-- DRIVER: pgx
ALTER TABLE my_noun
	ADD COLUMN my_field_integer BIGINT NOT NULL DEFAULT 0,
	ADD COLUMN my_field_nullable TEXT NULL,
	ADD COLUMN my_field_time TIMESTAMP WITH TIME ZONE NULL,
	ADD COLUMN my_field_tags BYTEA NULL,
	ADD COLUMN parent_table_id BIGINT NOT NULL DEFAULT 0;

-- DRIVER: mssql
ALTER TABLE my_noun ADD
	my_field_integer BIGINT NOT NULL DEFAULT 0,
	my_field_nullable NVARCHAR(MAX) NULL,
	my_field_time DATETIME2 NULL,
	my_field_tags VARBINARY(MAX) NULL,
	parent_table_id BIGINT NOT NULL DEFAULT 0;

-- DRIVER: sqlite
ALTER TABLE my_noun ADD COLUMN my_field_integer INTEGER NOT NULL DEFAULT 0;
-- DRIVER: sqlite
ALTER TABLE my_noun ADD COLUMN my_field_nullable TEXT;
-- DRIVER: sqlite
ALTER TABLE my_noun ADD COLUMN my_field_time DATETIME;
-- DRIVER: sqlite
ALTER TABLE my_noun ADD COLUMN my_field_tags BLOB;
-- DRIVER: sqlite
ALTER TABLE my_noun ADD COLUMN parent_table_id INTEGER NOT NULL DEFAULT 0;
```

**IMPORTANT**: SQLite only supports adding one column per `ALTER TABLE` statement. Each column must be a separate statement prefixed with `-- DRIVER: sqlite`.

Append `CREATE INDEX` or `CREATE UNIQUE INDEX` statements to add indices for columns that will be heavily searchable. Always include the `tenant_id` as the first column in a composite index. Name the index by concatenating the name of the table, followed by `idx` and the columns it includes (excluding the `tenant_id` column). For example, `my_noun_idx_my_field_integer` is a composite index of `(tenant_id, my_field_integer)` in the `my_noun` table. If you are not sure what columns are worth indexing, ask the user for guidance.

```sql
-- DRIVER: mysql
CREATE INDEX my_noun_idx_my_field_integer ON my_noun (tenant_id, my_field_integer);

-- DRIVER: pgx
CREATE INDEX my_noun_idx_my_field_integer ON my_noun (tenant_id, my_field_integer);

-- DRIVER: mssql
CREATE INDEX my_noun_idx_my_field_integer ON my_noun (tenant_id, my_field_integer);

-- DRIVER: sqlite
CREATE INDEX my_noun_idx_my_field_integer ON my_noun (tenant_id, my_field_integer);
```

**Production tables on Postgres**: plain `CREATE INDEX` takes an `ACCESS EXCLUSIVE` lock that blocks writes for the duration of the build - fine for small tables, painful for large hot ones. If the target `pgx` table has significant production write traffic, ask the user whether to use `CREATE INDEX CONCURRENTLY IF NOT EXISTS` instead. It builds without blocking writes (sequel runs each migration statement outside a transaction, so `CONCURRENTLY` is allowed), at the cost of: (a) two table scans instead of one, and (b) non-atomic failure mode - if interrupted, Postgres leaves an `INVALID` index that must be `DROP`-ed manually before the migration can be retried. Plain `CREATE INDEX` is the safer default for prototyping and for tables that are still small.

#### Step 5: Map Column Names to Object Fields

Update the mapping of the database column names to their corresponding object fields in `service.go`.
**IMPORTANT** : Do not remove the mappings of the `example` column to the `Example` field since they are required by various tests.

In `mapColumnsOnInsert`, map the column names that can be set during the initial insertion of the object.
For nullable columns, wrap the value in `sequel.Nullify` to store the Go zero value as `NULL` in the database. To use a SQL statement as value, wrap a string in `sequel.UnsafeSQL`.

```go
func (svc *Service) mapColumnsOnInsert(ctx context.Context, obj *serviceapi.Obj) (columnMapping map[string]any, err error) {
	tags, err := json.Marshal(obj.Tags)
	if err != nil {
		return errors.Trace(err)
	}
	columnMapping := map[string]any{
		"my_field_integer":  obj.MyFieldInteger,
		"my_field_nullable": sequel.Nullify(obj.MyFieldNullable),
		"my_field_time":     sequel.UnsafeSQL(svc.db.NowUTC()),
		"my_field_tags":     tags,
		"parent_table_id":   obj.ParentKey.ID,
	}
	return columnMapping, nil
}
```

In `mapColumnsOnUpdate`, map the columns that can be modified after the initial insertion of the object.
For nullable columns, wrap the value in `sequel.Nullify` to store the Go zero value as `NULL` in the database. To use a SQL statement as value, wrap a string in `sequel.UnsafeSQL`.

```go
func (svc *Service) mapColumnsOnUpdate(ctx context.Context, obj *serviceapi.Obj) (columnMapping map[string]any, err error) {
	tags, err := json.Marshal(obj.Tags)
	if err != nil {
		return errors.Trace(err)
	}
	columnMapping := map[string]any{
		"my_field_integer":  obj.MyFieldInteger,
		"my_field_nullable": sequel.Nullify(obj.MyFieldNullable),
		"my_field_time":     sequel.UnsafeSQL(svc.db.NowUTC()),
		"my_field_tags":     tags,
		"parent_table_id":   obj.ParentKey.ID,
	}
	return columnMapping, nil
}
```

In `mapColumnsOnSelect`, map the columns that can be read.
For nullable columns, wrap the reference to the variable in `sequel.Nullable` in order to interpret a database `NULL` value as the zero value of the Go data type. Use `sequel.Bind` to transform and apply the value manually to the object.

```go
func (svc *Service) mapColumnsOnSelect(ctx context.Context, obj *serviceapi.Obj) (columnMapping map[string]any, err error) {
	columnMapping := map[string]any{
		"my_field_integer":  &obj.MyFieldInteger,
		"my_field_nullable": sequel.Nullable(&obj.MyFieldNullable),
		"my_field_time":     &obj.MyFieldTime,
		"my_field_tags": sequel.Bind(func(value []byte) (err error) {
			return json.Unmarshal(value, &obj.Tags)
		}),
		"parent_table_id": &obj.ParentKey.ID,
	}
	return columnMapping, nil
}
```

#### Step 6: Add Query Conditions

Prepare appropriate query conditions in `prepareWhereClauses` in `service.go` for new `Query` fields. Only add a condition if the `Query` field is not its zero value. Add the names of any textual and searchable columns to the `searchableColumns` array.

```go
func (svc *Service) prepareWhereClauses(ctx context.Context, query serviceapi.Query) (conditions []string, args []any, err error) {
	if strings.TrimSpace(query.Q) != "" {
		searchableColumns := []string{
			"my_field_nullable",
		}
		// ...
	}
	// ...
	if query.MyFieldInteger != 0 {
		conditions = append(conditions,"my_field_integer=?")
		args = append(args, query.MyFieldInteger)
	}
	query.MyFieldNullable = strings.TrimSpace(query.MyFieldNullable)
	if query.MyFieldNullable != "" {
		conditions = append(conditions,"my_field_nullable=?")
		args = append(args, query.MyFieldNullable)
	}
	if !query.MyFieldTimeGTE.IsZero() {
		conditions = append(conditions,"my_field_time>=?")
		args = append(args, query.MyFieldTimeGTE)
	}
	if !query.MyFieldTimeLT.IsZero() {
		conditions = append(conditions,"my_field_time<?")
		args = append(args, query.MyFieldTimeLT)
	}
	if !query.ParentKey.IsZero() {
		conditions = append(conditions,"parent_table_id=?")
		args = append(args, query.ParentKey.ID)
	}
	return conditions, args, nil
}
```

#### Step 7: Update Integration Tests

The `NewObject` function in `service_test.go` is used by tests to construct a new object to pass to `Create`. Adjust the constructor function to initialize all required fields so that they pass validation. You may introduce a measure of randomness.

Extend the integration tests to take into account the schema changes. Look for the `HINT`s to guide you. In particular:
- Set, modify and verify the new fields in `TestMyNoun_ColumnMappings` in `service_test.go`
- Add validation test cases for new object fields in `TestMyNoun_ValidateObject` in `mynounapi/object_test.go`
- Add validation test cases for new query fields in `TestMyNoun_ValidateQuery` in `mynounapi/query_test.go`

#### Step 8: Housekeeping

Follow the `housekeeping` skill. Adding fields changes only the hand-written domain code (`object.go`, `query.go`, `service.go`, `resources/sql`), not `definition.go`, so the boilerplate regeneration is a no-op; the `Version` bump and `go vet` still apply.
