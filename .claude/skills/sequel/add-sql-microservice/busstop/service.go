package busstop

import (
	"context"
	"io/fs"
	"maps"
	"net/http"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/microbus-io/errors"
	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/frame"
	"github.com/microbus-io/dwarf/workflow"
	"github.com/microbus-io/sequel"

	"github.com/microbus-io/fabric/busstop/busstopapi"
)

var (
	_ context.Context
	_ http.Request
	_ errors.TracedError
	_ *workflow.Flow
	_ busstopapi.Client
)

const (
	tableName        = "bus_stop"
	sequenceName     = "bus_stop@_SEQUENCE_" // Do not change
	bulkBatchSize    = 1000
	maxParamsInQuery = 2000 // SQL Server is limited to 2100 parameters
)

/*
Service implements the microservice which persists bus stops in a SQL database.
*/
type Service struct {
	*Intermediate // IMPORTANT: Do not remove

	db *sequel.DB
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	err = svc.openDatabase(ctx)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	svc.closeDatabase(ctx)
	return nil
}

/*
mapColumnsOnInsert maps names of columns to their corresponding values, on creation of a new object.
*/
func (svc *Service) mapColumnsOnInsert(ctx context.Context, obj *busstopapi.BusStop) (columnMapping map[string]any, err error) {
	_ = ctx
	/*
		HINT: Map columns names to their corresponding values.
		Do not include columns that the actor is unauthorized to write to.
		Wrap a value in sequel.Nullify to store NULL in the database for its zero value.
		Wrap a string in sequel.UnsafeSQL to use a SQL statement as value.
	*/
	columnMapping = map[string]any{
		"example": sequel.Nullify(obj.Example), // Do not remove the example
	}
	return columnMapping, nil
}

/*
mapColumnsOnUpdate maps names of columns to their corresponding values, on modification of an already existing object.
*/
func (svc *Service) mapColumnsOnUpdate(ctx context.Context, obj *busstopapi.BusStop) (columnMapping map[string]any, err error) {
	_ = ctx
	/*
		HINT: Map columns names to their corresponding values.
		Do not include invariant columns.
		Do not include columns that the actor is unauthorized to write to.
		Wrap a value in sequel.Nullify to store NULL in the database for its zero value.
		Wrap a string in sequel.UnsafeSQL to use a SQL statement as value.
	*/
	columnMapping = map[string]any{
		"example": sequel.Nullify(obj.Example), // Do not remove the example
	}
	return columnMapping, nil
}

/*
mapColumnsOnSelect maps names of columns to their corresponding object fields, on reading of an object.
*/
func (svc *Service) mapColumnsOnSelect(ctx context.Context, obj *busstopapi.BusStop) (columnMapping map[string]any, err error) {
	_ = ctx
	/*
		HINT: Map columns names to their object fields.
		Exclude columns that the actor is unauthorized to read.
		Wrap the reference to a field in sequel.Nullable if the corresponding database column is NULL-able.
		Use sequel.Bind to transform and apply the value manually to the object.
	*/
	columnMapping = map[string]any{
		"example": sequel.Nullable(&obj.Example), // Do not remove the example
	}
	return columnMapping, nil
}

/*
prepareWhereClauses prepares the conditions to add to the WHERE clause based on the query.
All conditions must be met for a record to match the query, that is, they are AND-ed together.
*/
func (svc *Service) prepareWhereClauses(ctx context.Context, query busstopapi.Query) (conditions []string, args []any, err error) {
	_ = ctx
	if strings.TrimSpace(query.Q) != "" {
		searchableColumns := []string{
			/*
				HINT: Add names of textual (VARCHAR, TEXT, etc.) columns that are searchable.
				Exclude columns that the actor is unauthorized to search on.
			*/
			"example", // Do not remove the example
		}
		q := strings.TrimSpace(regexp.MustCompile(`\s`).ReplaceAllString(query.Q, " ")) // Compress whitespaces
		for _, word := range strings.Split(q, " ") {
			conditions = append(conditions, svc.db.RegexpTextSearch(searchableColumns...))
			if len([]rune(word)) <= 3 {
				args = append(args, `(^|\b)`+regexp.QuoteMeta(word))
			} else {
				args = append(args, regexp.QuoteMeta(word))
			}
		}
	}
	/*
		HINT: Add WHERE conditions for each non-zero filtering option of the query.
		Exclude columns that the actor is unauthorized to filter on.
	*/
	query.Example = strings.TrimSpace(query.Example) // Do not remove the example
	if query.Example != "" {
		conditions = append(conditions, "example=?")
		args = append(args, query.Example)
	}
	return conditions, args, nil
}

/*
tenantOf returns the tenant claim of the actor, or 0 if not found.
*/
func (svc *Service) tenantOf(ctx context.Context) int {
	tid, _ := frame.Of(ctx).Tenant()
	return tid
}

/*
openDatabase opens the database connection and runs schema migrations as needed.
*/
func (svc *Service) openDatabase(ctx context.Context) (err error) {
	_ = ctx
	const driverName = "" // The driver name is inferred from the data source name
	dataSourceName := svc.SQLDataSourceName()
	if dataSourceName == "" && svc.Deployment() == connector.LOCAL {
		dataSourceName = "file:local.sqlite"
	}
	if svc.Deployment() == connector.TESTING {
		dataSourceName, err = sequel.CreateTestingDatabase(driverName, dataSourceName, svc.Plane())
		if err != nil {
			return errors.Trace(err)
		}
	}
	svc.db, err = sequel.OpenSingleton(driverName, dataSourceName)
	if err != nil {
		return errors.Trace(err)
	}
	// Route sequel's spans, sequel_* metrics, and migration logs through the connector's telemetry pipeline.
	svc.db.SetTracerProvider(svc.TracerProvider())
	svc.db.SetMeterProvider(svc.MeterProvider())
	svc.db.SetLogger(svc.Logger())
	dirFS, err := fs.Sub(svc.ResFS(), "sql")
	if err != nil {
		return errors.Trace(err)
	}
	err = svc.db.Migrate(sequenceName, dirFS)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

/*
closeDatabase closes the database connection.
*/
func (svc *Service) closeDatabase(ctx context.Context) (err error) {
	_ = ctx
	if svc.db != nil {
		err = svc.db.Close()
	}
	return errors.Trace(err)
}

/*
Create creates a new object, returning its key.
*/
func (svc *Service) Create(ctx context.Context, obj *busstopapi.BusStop) (objKey busstopapi.BusStopKey, err error) { // MARKER: Create
	objKeys, err := svc.BulkCreate(ctx, []*busstopapi.BusStop{obj})
	if err != nil {
		return busstopapi.BusStopKey{}, errors.Trace(err)
	}
	return objKeys[0], nil
}

/*
Store updates the object.
*/
func (svc *Service) Store(ctx context.Context, obj *busstopapi.BusStop) (stored bool, err error) { // MARKER: Store
	storedKeys, err := svc.BulkStore(ctx, []*busstopapi.BusStop{obj})
	return len(storedKeys) > 0, errors.Trace(err)
}

/*
MustStore updates the object.
*/
func (svc *Service) MustStore(ctx context.Context, obj *busstopapi.BusStop) (err error) { // MARKER: MustStore
	stored, err := svc.Store(ctx, obj)
	if err != nil {
		return errors.Trace(err)
	}
	if !stored {
		return errors.New("object not found", http.StatusNotFound)
	}
	return nil
}

/*
Revise updates the object only if the revision matches.
*/
func (svc *Service) Revise(ctx context.Context, obj *busstopapi.BusStop) (revised bool, err error) { // MARKER: Revise
	revisedKeys, err := svc.BulkRevise(ctx, []*busstopapi.BusStop{obj})
	return len(revisedKeys) > 0, errors.Trace(err)
}

/*
MustRevise updates the object only if the revision matches.
*/
func (svc *Service) MustRevise(ctx context.Context, obj *busstopapi.BusStop) (err error) { // MARKER: MustRevise
	revised, err := svc.Revise(ctx, obj)
	if err != nil {
		return errors.Trace(err)
	}
	if !revised {
		return errors.New("revision conflict", http.StatusConflict)
	}
	return nil
}

/*
Delete deletes the object.
*/
func (svc *Service) Delete(ctx context.Context, objKey busstopapi.BusStopKey) (deleted bool, err error) { // MARKER: Delete
	deletedKeys, err := svc.BulkDelete(ctx, []busstopapi.BusStopKey{objKey})
	return len(deletedKeys) > 0, errors.Trace(err)
}

/*
MustDelete deletes the object.
*/
func (svc *Service) MustDelete(ctx context.Context, objKey busstopapi.BusStopKey) (err error) { // MARKER: MustDelete
	deleted, err := svc.Delete(ctx, objKey)
	if err != nil {
		return errors.Trace(err)
	}
	if !deleted {
		return errors.New("object not found", http.StatusNotFound)
	}
	return nil
}

/*
List returns the objects matching the query, and the total count of matches regardless of the limit.
*/
func (svc *Service) List(ctx context.Context, query busstopapi.Query) (objs []*busstopapi.BusStop, totalCount int, err error) { // MARKER: List
	err = query.Validate(ctx)
	if err != nil {
		return nil, 0, errors.Trace(err, http.StatusBadRequest)
	}
	var obj busstopapi.BusStop
	columnMapping, err := svc.mapColumnsOnSelect(ctx, &obj)
	if err != nil {
		return nil, 0, errors.Trace(err)
	}
	fuzzyColumnNames := map[string]string{"key": "id"}
	for k := range columnMapping {
		fuzzyColumnNames[strings.ToLower(strings.ReplaceAll(k, "_", ""))] = k
	}

	// -- SELECT --
	tenantID := svc.tenantOf(ctx)
	columnMapping["id"] = &obj.Key
	columnMapping["revision"] = &obj.Revision
	columnMapping["created_at"] = &obj.CreatedAt
	columnMapping["updated_at"] = &obj.UpdatedAt
	var columnsInOrder []string
	if query.Select != "" {
		for _, col := range strings.Split(query.Select, ",") {
			col = strings.TrimSpace(col)
			if _, ok := columnMapping[col]; !ok {
				// Fuzzy match to column_name
				col = fuzzyColumnNames[strings.ToLower(col)]
			}
			if _, ok := columnMapping[col]; ok && col != "" {
				columnsInOrder = append(columnsInOrder, col)
			}
		}
		sort.Strings(columnsInOrder)
	}
	if len(columnsInOrder) == 0 {
		columnsInOrder = slices.Sorted(maps.Keys(columnMapping))
	}
	var stmt strings.Builder
	stmt.WriteString("SELECT ")
	stmt.WriteString(strings.Join(columnsInOrder, ", "))
	stmt.WriteString(" FROM ")
	stmt.WriteString(tableName)
	stmt.WriteString(" WHERE tenant_id=?")
	args := []any{
		tenantID,
	}

	// -- WHERE --
	if !query.Key.IsZero() {
		stmt.WriteString(" AND id=?")
		args = append(args, query.Key)
	}
	if len(query.Keys) > 0 {
		stmt.WriteString(" AND id IN (")
		for i, k := range query.Keys {
			if i > 0 {
				stmt.WriteString(",")
			}
			stmt.WriteString(strconv.Itoa(k.ID))
		}
		stmt.WriteString(")")
	} else if query.Keys != nil {
		stmt.WriteString(" AND 1=0")
	}
	conditions, conditionArgs, err := svc.prepareWhereClauses(ctx, query)
	if err != nil {
		return nil, 0, errors.Trace(err)
	}
	for _, cond := range conditions {
		stmt.WriteString(" AND (")
		stmt.WriteString(cond)
		stmt.WriteString(")")
	}
	args = append(args, conditionArgs...)

	// -- ORDER BY --
	countOrderBy := 0
	orderedByID := false
	stmt.WriteString(" ORDER BY")
	for orderBy := range strings.SplitSeq(query.OrderBy, ",") {
		orderBy := strings.TrimSpace(strings.ToLower(orderBy))
		if orderBy == "" {
			continue
		}
		direction := "ASC"
		if strings.HasPrefix(orderBy, "-") {
			orderBy = orderBy[1:]
			direction = "DESC"
		}
		if _, ok := columnMapping[orderBy]; !ok {
			// Fuzzy match to column_name
			orderBy = fuzzyColumnNames[strings.ToLower(orderBy)]
		}
		if _, ok := columnMapping[orderBy]; !ok {
			continue
		}
		if orderBy == "example" && svc.Deployment() != connector.TESTING {
			continue
		}
		if orderBy == "id" {
			orderedByID = true
		}
		if countOrderBy > 0 {
			stmt.WriteString(",")
		}
		countOrderBy++
		stmt.WriteString(" ")
		stmt.WriteString(orderBy)
		stmt.WriteString(" ")
		stmt.WriteString(direction)
	}
	if !orderedByID {
		if countOrderBy > 0 {
			stmt.WriteString(",")
		}
		stmt.WriteString(" id ASC")
	}

	// -- OFFSET/LIMIT --
	countArgsBeforeOffsetLimit := len(args)
	if (query.Offset > 0 || query.Limit > 0) && query.Offset >= 0 && query.Limit >= 0 {
		limit := query.Limit
		if limit == 0 {
			limit = 1 << 62 // Infinity
		}
		stmt.WriteString(" LIMIT_OFFSET(?, ?)")
		args = append(args, limit, query.Offset)
	}

	// Query
	stmtStr := stmt.String()
	f1 := func(ctx context.Context) (err error) {
		// Query for the objects
		rows, err := svc.db.QueryContext(ctx, stmtStr, args...)
		if err != nil {
			return errors.Trace(err)
		}
		defer rows.Close()
		scanArgs := []any{}
		for _, k := range columnsInOrder {
			scanArgs = append(scanArgs, columnMapping[k])
		}
		testing := svc.Deployment() == connector.TESTING
		for rows.Next() {
			err := rows.Scan(scanArgs...)
			if err != nil {
				return errors.Trace(err)
			}
			err = sequel.ApplyBindings(scanArgs...)
			if err != nil {
				return errors.Trace(err)
			}
			copy := obj
			if !testing {
				copy.Example = ""
			}
			objs = append(objs, &copy)
		}
		return nil
	}
	f2 := func(ctx context.Context) (err error) {
		// Query for the total count
		p := strings.Index(stmtStr, "FROM "+tableName+" WHERE ")
		q := strings.Index(stmtStr, "ORDER BY ")
		err = svc.db.QueryRowContext(ctx, "SELECT COUNT(*) "+stmtStr[p:q], args[:countArgsBeforeOffsetLimit]...).Scan(&totalCount)
		return errors.Trace(err)
	}
	if !query.Key.IsZero() || (query.Offset == 0 && query.Limit == 0) {
		// No need to count separately when fetching by key or when fetching the entire dataset
		err = f1(ctx)
		totalCount = len(objs)
	} else {
		err = svc.Parallel(ctx, f1, f2)
	}
	if err != nil {
		return nil, 0, errors.Trace(err)
	}
	return objs, totalCount, nil
}

/*
Lookup returns the single object matching the query. It errors if more than one object matches the query.
*/
func (svc *Service) Lookup(ctx context.Context, query busstopapi.Query) (obj *busstopapi.BusStop, found bool, err error) { // MARKER: Lookup
	err = query.Validate(ctx)
	if err != nil {
		return nil, false, errors.Trace(err, http.StatusBadRequest)
	}
	query.Offset = 0
	query.Limit = 2
	objs, _, err := svc.List(ctx, query)
	if err != nil {
		return nil, false, errors.Trace(err)
	}
	switch len(objs) {
	case 0:
		return nil, false, nil
	case 1:
		return objs[0], true, nil
	default:
		return nil, false, errors.New("more than one object matched the query")
	}
}

/*
MustLookup returns the single object matching the query. It errors unless exactly one object matches the query.
*/
func (svc *Service) MustLookup(ctx context.Context, query busstopapi.Query) (obj *busstopapi.BusStop, err error) { // MARKER: MustLookup
	obj, found, err := svc.Lookup(ctx, query)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if !found {
		return nil, errors.New("object not found", http.StatusNotFound)
	}
	return obj, nil
}

/*
Load returns the object associated with the key.
*/
func (svc *Service) Load(ctx context.Context, objKey busstopapi.BusStopKey) (obj *busstopapi.BusStop, found bool, err error) { // MARKER: Load
	if objKey.IsZero() {
		return nil, false, nil
	}
	obj, found, err = svc.Lookup(ctx, busstopapi.Query{Key: objKey})
	return obj, found, errors.Trace(err)
}

/*
MustLoad returns the object associated with the key. It errors if the object is not found.
*/
func (svc *Service) MustLoad(ctx context.Context, objKey busstopapi.BusStopKey) (obj *busstopapi.BusStop, err error) { // MARKER: MustLoad
	obj, ok, err := svc.Load(ctx, objKey)
	if err != nil {
		return nil, errors.Trace(err)
	}
	if !ok {
		return nil, errors.New("object not found", http.StatusNotFound)
	}
	return obj, nil
}

/*
BulkLoad returns the objects matching the keys.
*/
func (svc *Service) BulkLoad(ctx context.Context, objKeys []busstopapi.BusStopKey) (objs []*busstopapi.BusStop, err error) { // MARKER: BulkLoad
	if len(objKeys) == 0 {
		return nil, nil
	}
	sort.Slice(objKeys, func(i, j int) bool {
		return objKeys[i].ID < objKeys[j].ID
	})
	for len(objKeys) > 0 {
		n := len(objKeys)
		var batch []*busstopapi.BusStop
		if n <= bulkBatchSize {
			batch, _, err = svc.List(ctx, busstopapi.Query{
				Keys: objKeys,
			})
			objKeys = nil
		} else {
			batch, _, err = svc.List(ctx, busstopapi.Query{
				Keys: objKeys[:bulkBatchSize],
			})
			objKeys = objKeys[bulkBatchSize:]
		}
		if err != nil {
			return nil, errors.Trace(err)
		}
		objs = append(objs, batch...)
	}
	return objs, nil
}

/*
BulkDelete deletes the objects matching the keys, returning the keys of the deleted objects.
*/
func (svc *Service) BulkDelete(ctx context.Context, objKeys []busstopapi.BusStopKey) (deletedKeys []busstopapi.BusStopKey, err error) { // MARKER: BulkDelete
	if len(objKeys) == 0 {
		return nil, nil
	}
	// Sort by ID to optimize disk access
	sort.Slice(objKeys, func(i, j int) bool {
		return objKeys[i].ID < objKeys[j].ID
	})
	tenantID := svc.tenantOf(ctx)
	for len(objKeys) > 0 {
		var batch []busstopapi.BusStopKey
		if len(objKeys) <= bulkBatchSize {
			batch = objKeys
			objKeys = nil
		} else {
			batch = objKeys[:bulkBatchSize]
			objKeys = objKeys[bulkBatchSize:]
		}
		writeIDList := func(stmt *strings.Builder) {
			for i, k := range batch {
				if i > 0 {
					stmt.WriteString(",")
				}
				stmt.WriteString(strconv.Itoa(k.ID))
			}
		}
		switch svc.db.DriverName() {
		case "mysql":
			// MySQL doesn't support RETURNING; use a transaction with SELECT FOR UPDATE
			tx, err := svc.db.BeginTx(ctx, nil)
			if err != nil {
				return deletedKeys, errors.Trace(err)
			}
			var selectStmt strings.Builder
			selectStmt.WriteString("SELECT id FROM ")
			selectStmt.WriteString(tableName)
			selectStmt.WriteString(" WHERE tenant_id=? AND id IN (")
			writeIDList(&selectStmt)
			selectStmt.WriteString(") FOR UPDATE")
			rows, err := tx.QueryContext(ctx, selectStmt.String(), tenantID)
			if err != nil {
				tx.Rollback()
				return deletedKeys, errors.Trace(err)
			}
			var foundKeys []busstopapi.BusStopKey
			for rows.Next() {
				var key busstopapi.BusStopKey
				err = rows.Scan(&key)
				if err != nil {
					rows.Close()
					tx.Rollback()
					return deletedKeys, errors.Trace(err)
				}
				foundKeys = append(foundKeys, key)
			}
			rows.Close()
			if len(foundKeys) > 0 {
				var deleteStmt strings.Builder
				deleteStmt.WriteString("DELETE FROM ")
				deleteStmt.WriteString(tableName)
				deleteStmt.WriteString(" WHERE tenant_id=? AND id IN (")
				writeIDList(&deleteStmt)
				deleteStmt.WriteString(")")
				_, err = tx.ExecContext(ctx, deleteStmt.String(), tenantID)
				if err != nil {
					tx.Rollback()
					return deletedKeys, errors.Trace(err)
				}
			}
			err = tx.Commit()
			if err != nil {
				return deletedKeys, errors.Trace(err)
			}
			deletedKeys = append(deletedKeys, foundKeys...)
		case "pgx", "sqlite":
			// PostgreSQL supports RETURNING
			var stmt strings.Builder
			stmt.WriteString("DELETE FROM ")
			stmt.WriteString(tableName)
			stmt.WriteString(" WHERE tenant_id=? AND id IN (")
			writeIDList(&stmt)
			stmt.WriteString(") RETURNING id")
			rows, err := svc.db.QueryContext(ctx, stmt.String(), tenantID)
			if err != nil {
				return deletedKeys, errors.Trace(err)
			}
			for rows.Next() {
				var key busstopapi.BusStopKey
				err = rows.Scan(&key)
				if err != nil {
					rows.Close()
					return deletedKeys, errors.Trace(err)
				}
				deletedKeys = append(deletedKeys, key)
			}
			rows.Close()
		case "mssql":
			// SQL Server supports OUTPUT DELETED
			var stmt strings.Builder
			stmt.WriteString("DELETE FROM ")
			stmt.WriteString(tableName)
			stmt.WriteString(" OUTPUT DELETED.id")
			stmt.WriteString(" WHERE tenant_id=? AND id IN (")
			writeIDList(&stmt)
			stmt.WriteString(")")
			rows, err := svc.db.QueryContext(ctx, stmt.String(), tenantID)
			if err != nil {
				return deletedKeys, errors.Trace(err)
			}
			for rows.Next() {
				var key busstopapi.BusStopKey
				err = rows.Scan(&key)
				if err != nil {
					rows.Close()
					return deletedKeys, errors.Trace(err)
				}
				deletedKeys = append(deletedKeys, key)
			}
			rows.Close()
		}
	}
	// Fire-and-forget event
	if len(deletedKeys) > 0 {
		busstopapi.NewMulticastTrigger(svc).OnBusStopDeleted(ctx, deletedKeys)
	}
	return deletedKeys, nil
}

/*
BulkStore updates multiple objects, returning the keys of the stored objects.
*/
func (svc *Service) BulkStore(ctx context.Context, objs []*busstopapi.BusStop) (storedKeys []busstopapi.BusStopKey, err error) { // MARKER: BulkStore
	storedKeys, err = svc.bulkUpdate(ctx, objs, false)
	if err != nil {
		return storedKeys, errors.Trace(err)
	}
	// Fire-and-forget event
	if len(storedKeys) > 0 {
		busstopapi.NewMulticastTrigger(svc).OnBusStopStored(ctx, storedKeys)
	}
	return storedKeys, nil
}

/*
BulkRevise updates multiple objects, returning the number of rows affected.
Only rows with matching revisions are updated.
*/
func (svc *Service) BulkRevise(ctx context.Context, objs []*busstopapi.BusStop) (revisedKeys []busstopapi.BusStopKey, err error) { // MARKER: BulkRevise
	revisedKeys, err = svc.bulkUpdate(ctx, objs, true)
	return revisedKeys, errors.Trace(err)
}

// bulkUpdate implements [Service.BulkStore] and [Service.BulkRevise].
func (svc *Service) bulkUpdate(ctx context.Context, objs []*busstopapi.BusStop, checkRevision bool) (storedKeys []busstopapi.BusStopKey, err error) {
	if len(objs) == 0 {
		return nil, nil
	}
	// Validate all objects before updating any
	for i, obj := range objs {
		if obj == nil {
			return nil, errors.New("nil object", http.StatusBadRequest, "index", i)
		}
		if obj.Key.IsZero() {
			return nil, errors.New("zero key", http.StatusBadRequest, "index", i)
		}
		err = obj.Validate(ctx)
		if err != nil {
			return nil, errors.Trace(err, http.StatusBadRequest, "index", i)
		}
	}
	// Sort by ID to optimize disk access
	sort.Slice(objs, func(i, j int) bool {
		return objs[i].Key.ID < objs[j].Key.ID
	})
	testing := svc.Deployment() == connector.TESTING
	tenantID := svc.tenantOf(ctx)

	// Determine column order and params per row from the first object
	if !testing {
		objs[0].Example = ""
	}
	firstMapping, err := svc.mapColumnsOnUpdate(ctx, objs[0])
	if err != nil {
		return nil, errors.Trace(err)
	}
	delete(firstMapping, "tenant_id")
	delete(firstMapping, "revision")
	delete(firstMapping, "created_at")
	firstMapping["updated_at"] = sequel.UnsafeSQL("NOW_UTC()")
	columnsInOrder := slices.Sorted(maps.Keys(firstMapping))
	paramsPerRow := 0
	for _, k := range columnsInOrder {
		if _, ok := firstMapping[k].(sequel.UnsafeSQL); ok {
			continue
		}
		if _, ok := firstMapping[k].(int); ok {
			continue
		}
		paramsPerRow++
	}
	if paramsPerRow == 0 {
		paramsPerRow = 1 // Avoid division by zero
	}
	batchSize := max(maxParamsInQuery/paramsPerRow, 1)
	batchSize = min(batchSize, bulkBatchSize)

	// Process in batches
	for batchStart := 0; batchStart < len(objs); batchStart += batchSize {
		batchEnd := min(batchStart+batchSize, len(objs))
		batch := objs[batchStart:batchEnd]

		// Build column mappings for this batch
		batchMappings := make([]map[string]any, len(batch))
		for i, obj := range batch {
			if batchStart == 0 && i == 0 {
				batchMappings[i] = firstMapping
			} else {
				if !testing {
					obj.Example = ""
				}
				batchMappings[i], err = svc.mapColumnsOnUpdate(ctx, obj)
				if err != nil {
					return nil, errors.Trace(err)
				}
				delete(batchMappings[i], "tenant_id")
				delete(batchMappings[i], "revision")
				delete(batchMappings[i], "created_at")
				batchMappings[i]["updated_at"] = sequel.UnsafeSQL("NOW_UTC()")
			}
		}

		// Build CASE UPDATE statement
		var stmt strings.Builder
		stmt.WriteString("UPDATE ")
		stmt.WriteString(tableName)
		stmt.WriteString(" SET ")
		args := []any{}
		for _, col := range columnsInOrder {
			stmt.WriteString(col)
			stmt.WriteString("=")
			if len(batchMappings) > 1 {
				stmt.WriteString("CASE")
				for i, mapping := range batchMappings {
					stmt.WriteString(" WHEN id=")
					stmt.WriteString(strconv.Itoa(batch[i].Key.ID))
					stmt.WriteString(" THEN ")
					v := mapping[col]
					if unsafeSQL, ok := v.(sequel.UnsafeSQL); ok {
						stmt.WriteString("(")
						stmt.WriteString(string(unsafeSQL))
						stmt.WriteString(")")
					} else if vint, ok := v.(int); ok {
						stmt.WriteString(strconv.Itoa(vint))
					} else {
						stmt.WriteString("?")
						args = append(args, v)
					}
				}
				stmt.WriteString(" END")
			} else {
				v := batchMappings[0][col]
				if unsafeSQL, ok := v.(sequel.UnsafeSQL); ok {
					stmt.WriteString("(")
					stmt.WriteString(string(unsafeSQL))
					stmt.WriteString(")")
				} else if vint, ok := v.(int); ok {
					stmt.WriteString(strconv.Itoa(vint))
				} else {
					stmt.WriteString("?")
					args = append(args, v)
				}
			}
			stmt.WriteString(",")
		}
		stmt.WriteString("revision=(1+revision)")
		if svc.db.DriverName() == "mssql" {
			stmt.WriteString(" OUTPUT INSERTED.id")
		}
		stmt.WriteString(" WHERE tenant_id=")
		stmt.WriteString(strconv.Itoa(tenantID))
		if !checkRevision {
			stmt.WriteString(" AND id IN (")
			for i, obj := range batch {
				if i > 0 {
					stmt.WriteString(",")
				}
				stmt.WriteString(strconv.Itoa(obj.Key.ID))
			}
			stmt.WriteString(")")
		} else {
			stmt.WriteString(" AND (")
			for i, obj := range batch {
				if i > 0 {
					stmt.WriteString(" OR ")
				}
				stmt.WriteString("id=")
				stmt.WriteString(strconv.Itoa(obj.Key.ID))
				stmt.WriteString(" AND revision=")
				stmt.WriteString(strconv.Itoa(obj.Revision))
			}
			stmt.WriteString(")")
		}
		if svc.db.DriverName() == "pgx" || svc.db.DriverName() == "sqlite" {
			stmt.WriteString(" RETURNING id")
		}
		stmtStr := stmt.String()
		switch svc.db.DriverName() {
		case "mysql":
			// MySQL doesn't support RETURNING; use a transaction with SELECT FOR UPDATE
			tx, err := svc.db.BeginTx(ctx, nil)
			if err != nil {
				return storedKeys, errors.Trace(err)
			}
			var selectStmt strings.Builder
			selectStmt.WriteString("SELECT id FROM ")
			selectStmt.WriteString(tableName)
			selectStmt.WriteString(" WHERE tenant_id=?")
			if !checkRevision {
				selectStmt.WriteString(" AND id IN (")
				for i, obj := range batch {
					if i > 0 {
						selectStmt.WriteString(",")
					}
					selectStmt.WriteString(strconv.Itoa(obj.Key.ID))
				}
				selectStmt.WriteString(")")
			} else {
				selectStmt.WriteString(" AND (")
				for i, obj := range batch {
					if i > 0 {
						selectStmt.WriteString(" OR ")
					}
					selectStmt.WriteString("id=")
					selectStmt.WriteString(strconv.Itoa(obj.Key.ID))
					selectStmt.WriteString(" AND revision=")
					selectStmt.WriteString(strconv.Itoa(obj.Revision))
				}
				selectStmt.WriteString(")")
			}
			selectStmt.WriteString(" FOR UPDATE")
			rows, err := tx.QueryContext(ctx, selectStmt.String(), tenantID)
			if err != nil {
				tx.Rollback()
				return storedKeys, errors.Trace(err)
			}
			var foundKeys []busstopapi.BusStopKey
			for rows.Next() {
				var key busstopapi.BusStopKey
				err = rows.Scan(&key)
				if err != nil {
					rows.Close()
					tx.Rollback()
					return storedKeys, errors.Trace(err)
				}
				foundKeys = append(foundKeys, key)
			}
			rows.Close()
			if len(foundKeys) > 0 {
				_, err = tx.ExecContext(ctx, stmtStr, args...)
				if err != nil {
					tx.Rollback()
					return storedKeys, errors.Trace(err)
				}
			}
			err = tx.Commit()
			if err != nil {
				return storedKeys, errors.Trace(err)
			}
			storedKeys = append(storedKeys, foundKeys...)
		default:
			// pgx and mssql support RETURNING/OUTPUT
			rows, err := svc.db.QueryContext(ctx, stmtStr, args...)
			if err != nil {
				return storedKeys, errors.Trace(err)
			}
			for rows.Next() {
				var key busstopapi.BusStopKey
				err = rows.Scan(&key)
				if err != nil {
					rows.Close()
					return storedKeys, errors.Trace(err)
				}
				storedKeys = append(storedKeys, key)
			}
			rows.Close()
		}
	}
	return storedKeys, nil
}

/*
BulkCreate creates multiple objects, returning their keys.
*/
func (svc *Service) BulkCreate(ctx context.Context, objs []*busstopapi.BusStop) (objKeys []busstopapi.BusStopKey, err error) { // MARKER: BulkCreate
	if len(objs) == 0 {
		return nil, nil
	}
	// Validate all objects before inserting any
	for i, obj := range objs {
		if obj == nil {
			return nil, errors.New("nil object", http.StatusBadRequest, "index", i)
		}
		err = obj.Validate(ctx)
		if err != nil {
			return nil, errors.Trace(err, http.StatusBadRequest, "index", i)
		}
	}
	testing := svc.Deployment() == connector.TESTING
	tenantID := svc.tenantOf(ctx)

	// Determine column order and params per row from the first object
	if !testing {
		objs[0].Example = ""
	}
	firstMapping, err := svc.mapColumnsOnInsert(ctx, objs[0])
	if err != nil {
		return nil, errors.Trace(err)
	}
	sqlNow := sequel.UnsafeSQL("NOW_UTC()")
	firstMapping["tenant_id"] = tenantID
	firstMapping["revision"] = 1
	firstMapping["created_at"] = sqlNow
	firstMapping["updated_at"] = sqlNow
	firstMapping["reserved_before"] = sqlNow
	columnsInOrder := slices.Sorted(maps.Keys(firstMapping))
	paramsPerRow := 0
	for _, k := range columnsInOrder {
		if _, ok := firstMapping[k].(sequel.UnsafeSQL); ok {
			continue
		}
		if _, ok := firstMapping[k].(int); ok {
			continue
		}
		paramsPerRow++
	}
	if paramsPerRow == 0 {
		paramsPerRow = 1 // Avoid division by zero
	}
	batchSize := max(maxParamsInQuery/paramsPerRow, 1)
	batchSize = min(batchSize, bulkBatchSize)

	// Process in batches
	objKeys = make([]busstopapi.BusStopKey, 0, len(objs))
	for batchStart := 0; batchStart < len(objs); batchStart += batchSize {
		batchEnd := min(batchStart+batchSize, len(objs))

		// Build column mappings and multi-row INSERT statement for this batch
		var stmt strings.Builder
		stmt.WriteString("INSERT INTO ")
		stmt.WriteString(tableName)
		stmt.WriteString(" (")
		stmt.WriteString(strings.Join(columnsInOrder, ","))
		stmt.WriteString(")")
		args := []any{}
		switch svc.db.DriverName() {
		case "mssql":
			stmt.WriteString(" OUTPUT INSERTED.id")
		}
		stmt.WriteString(" VALUES ")
		for i, obj := range objs[batchStart:batchEnd] {
			// The first object was already mapped above
			var mapping map[string]any
			if batchStart == 0 && i == 0 {
				mapping = firstMapping
			} else {
				if !testing {
					obj.Example = ""
				}
				mapping, err = svc.mapColumnsOnInsert(ctx, obj)
				if err != nil {
					return nil, errors.Trace(err)
				}
				mapping["tenant_id"] = tenantID
				mapping["revision"] = 1
				mapping["created_at"] = sqlNow
				mapping["updated_at"] = sqlNow
				mapping["reserved_before"] = sqlNow
			}
			if i > 0 {
				stmt.WriteString(",")
			}
			stmt.WriteString("(")
			for j, k := range columnsInOrder {
				if j > 0 {
					stmt.WriteString(",")
				}
				v := mapping[k]
				if unsafeSQL, ok := v.(sequel.UnsafeSQL); ok {
					stmt.WriteString("(")
					stmt.WriteString(string(unsafeSQL))
					stmt.WriteString(")")
				} else if vint, ok := v.(int); ok {
					stmt.WriteString(strconv.Itoa(vint))
				} else {
					stmt.WriteString("?")
					args = append(args, v)
				}
			}
			stmt.WriteString(")")
		}
		switch svc.db.DriverName() {
		case "pgx", "sqlite":
			stmt.WriteString(" RETURNING id")
		}
		stmtStr := stmt.String()

		// Execute and retrieve generated IDs
		switch svc.db.DriverName() {
		case "mysql":
			res, err := svc.db.ExecContext(ctx, stmtStr, args...)
			if err != nil {
				return nil, errors.Trace(err)
			}
			firstID, err := res.LastInsertId()
			if err != nil {
				return nil, errors.Trace(err)
			}
			for i := range batchEnd - batchStart {
				objKeys = append(objKeys, busstopapi.BusStopKey{ID: int(firstID) + i})
			}
		default:
			rows, err := svc.db.QueryContext(ctx, stmtStr, args...)
			if err != nil {
				return nil, errors.Trace(err)
			}
			for rows.Next() {
				var key busstopapi.BusStopKey
				err = rows.Scan(&key)
				if err != nil {
					rows.Close()
					return nil, errors.Trace(err)
				}
				objKeys = append(objKeys, key)
			}
			rows.Close()
		}
	}
	// Fire-and-forget event
	if len(objKeys) > 0 {
		busstopapi.NewMulticastTrigger(svc).OnBusStopCreated(ctx, objKeys)
	}
	return objKeys, nil
}

/*
Purge deletes all objects matching the query, returning the keys of the deleted objects.
*/
func (svc *Service) Purge(ctx context.Context, query busstopapi.Query) (deletedKeys []busstopapi.BusStopKey, err error) { // MARKER: Purge
	query.Select = "id"
	objs, _, err := svc.List(ctx, query)
	if err != nil {
		return nil, errors.Trace(err)
	}
	keys := make([]busstopapi.BusStopKey, len(objs))
	for i, obj := range objs {
		keys[i] = obj.Key
	}
	deletedKeys, err = svc.BulkDelete(ctx, keys)
	return deletedKeys, errors.Trace(err)
}

/*
Count returns the number of objects matching the query, disregarding pagination.
*/
func (svc *Service) Count(ctx context.Context, query busstopapi.Query) (count int, err error) { // MARKER: Count
	query.Offset = 0
	query.Limit = 1
	query.Select = "id"
	_, count, err = svc.List(ctx, query)
	return count, errors.Trace(err)
}

/*
CreateREST creates a new bus stop via REST, returning its key.
*/
func (svc *Service) CreateREST(ctx context.Context, httpRequestBody *busstopapi.BusStop) (objKey busstopapi.BusStopKey, httpStatusCode int, err error) { // MARKER: CreateREST
	objKey, err = svc.Create(ctx, httpRequestBody)
	if err != nil {
		return objKey, http.StatusInternalServerError, errors.Trace(err)
	}
	return objKey, http.StatusCreated, nil
}

/*
StoreREST updates an existing bus stop via REST.
*/
func (svc *Service) StoreREST(ctx context.Context, key busstopapi.BusStopKey, httpRequestBody *busstopapi.BusStop) (httpStatusCode int, err error) { // MARKER: StoreREST
	httpRequestBody.Key = key
	stored, err := svc.Store(ctx, httpRequestBody)
	if err != nil {
		return http.StatusInternalServerError, errors.Trace(err)
	}
	if !stored {
		return http.StatusNotFound, nil
	}
	return http.StatusNoContent, nil
}

/*
DeleteREST deletes an existing bus stop via REST.
*/
func (svc *Service) DeleteREST(ctx context.Context, key busstopapi.BusStopKey) (httpStatusCode int, err error) { // MARKER: DeleteREST
	deleted, err := svc.Delete(ctx, key)
	if err != nil {
		return http.StatusInternalServerError, errors.Trace(err)
	}
	if !deleted {
		return http.StatusNotFound, nil
	}
	return http.StatusNoContent, nil
}

/*
LoadREST loads a bus stop by key via REST.
*/
func (svc *Service) LoadREST(ctx context.Context, key busstopapi.BusStopKey) (httpResponseBody *busstopapi.BusStop, httpStatusCode int, err error) { // MARKER: LoadREST
	obj, found, err := svc.Load(ctx, key)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.Trace(err)
	}
	if !found {
		return nil, http.StatusNotFound, nil
	}
	return obj, http.StatusOK, nil
}

/*
ListREST lists bus stops matching the query via REST.
*/
func (svc *Service) ListREST(ctx context.Context, q busstopapi.Query) (httpResponseBody []*busstopapi.BusStop, httpStatusCode int, err error) { // MARKER: ListREST
	objs, _, err := svc.List(ctx, q)
	if err != nil {
		return nil, http.StatusInternalServerError, errors.Trace(err)
	}
	return objs, http.StatusOK, nil
}

/*
TryReserve attempts to reserve a bus stop for the given duration, returning true if successful.
*/
func (svc *Service) TryReserve(ctx context.Context, objKey busstopapi.BusStopKey, dur time.Duration) (reserved bool, err error) { // MARKER: TryReserve
	reservedKeys, err := svc.TryBulkReserve(ctx, []busstopapi.BusStopKey{objKey}, dur)
	return len(reservedKeys) > 0, errors.Trace(err)
}

/*
TryBulkReserve attempts to reserve bus stops for the given duration, returning the keys of those successfully reserved.
Only bus stops whose reservation has expired (reserved_before < NOW) are reserved.
*/
func (svc *Service) TryBulkReserve(ctx context.Context, objKeys []busstopapi.BusStopKey, dur time.Duration) (reservedKeys []busstopapi.BusStopKey, err error) { // MARKER: TryBulkReserve
	return svc.bulkReserve(ctx, objKeys, dur, false)
}

/*
Reserve unconditionally reserves a bus stop for the given duration, returning true if the bus stop exists.
*/
func (svc *Service) Reserve(ctx context.Context, objKey busstopapi.BusStopKey, dur time.Duration) (reserved bool, err error) { // MARKER: Reserve
	reservedKeys, err := svc.BulkReserve(ctx, []busstopapi.BusStopKey{objKey}, dur)
	return len(reservedKeys) > 0, errors.Trace(err)
}

/*
BulkReserve unconditionally reserves bus stops for the given duration, returning the keys of those that exist.
*/
func (svc *Service) BulkReserve(ctx context.Context, objKeys []busstopapi.BusStopKey, dur time.Duration) (reservedKeys []busstopapi.BusStopKey, err error) { // MARKER: BulkReserve
	return svc.bulkReserve(ctx, objKeys, dur, true)
}

// bulkReserve is the shared implementation for TryBulkReserve and BulkReserve.
// When forceful is true, the reservation is set regardless of current state.
// When forceful is false, only rows where reserved_before <= NOW are reserved.
func (svc *Service) bulkReserve(ctx context.Context, objKeys []busstopapi.BusStopKey, dur time.Duration, forceful bool) (reservedKeys []busstopapi.BusStopKey, err error) {
	if len(objKeys) == 0 {
		return nil, nil
	}
	if dur < 0 {
		return nil, errors.New("duration must not be negative", http.StatusBadRequest)
	}
	if !forceful && dur == 0 {
		// TryReserve with duration 0 would only match already-expired reservations
		// and set them to NOW, which leaves them expired. No-op.
		return nil, nil
	}
	// Sort by ID to optimize disk access
	sort.Slice(objKeys, func(i, j int) bool {
		return objKeys[i].ID < objKeys[j].ID
	})
	tenantID := svc.tenantOf(ctx)
	durMillis := dur.Milliseconds()
	if dur > 0 && durMillis <= 0 {
		durMillis = 1
	}

	for len(objKeys) > 0 {
		var batch []busstopapi.BusStopKey
		if len(objKeys) <= bulkBatchSize {
			batch = objKeys
			objKeys = nil
		} else {
			batch = objKeys[:bulkBatchSize]
			objKeys = objKeys[bulkBatchSize:]
		}
		writeIDList := func(stmt *strings.Builder) {
			for i, k := range batch {
				if i > 0 {
					stmt.WriteString(",")
				}
				stmt.WriteString(strconv.Itoa(k.ID))
			}
		}

		switch svc.db.DriverName() {
		case "mysql":
			// MySQL doesn't support RETURNING; use a transaction with SELECT FOR UPDATE
			tx, err := svc.db.BeginTx(ctx, nil)
			if err != nil {
				return reservedKeys, errors.Trace(err)
			}
			var selectStmt strings.Builder
			selectStmt.WriteString("SELECT id FROM ")
			selectStmt.WriteString(tableName)
			selectStmt.WriteString(" WHERE tenant_id=? AND id IN (")
			writeIDList(&selectStmt)
			selectStmt.WriteString(")")
			if !forceful {
				selectStmt.WriteString(" AND reserved_before<=NOW_UTC()")
			}
			selectStmt.WriteString(" FOR UPDATE")
			rows, err := tx.QueryContext(ctx, selectStmt.String(), tenantID)
			if err != nil {
				tx.Rollback()
				return reservedKeys, errors.Trace(err)
			}
			var foundKeys []busstopapi.BusStopKey
			for rows.Next() {
				var key busstopapi.BusStopKey
				err = rows.Scan(&key)
				if err != nil {
					rows.Close()
					tx.Rollback()
					return reservedKeys, errors.Trace(err)
				}
				foundKeys = append(foundKeys, key)
			}
			rows.Close()
			if len(foundKeys) > 0 {
				var updateStmt strings.Builder
				updateStmt.WriteString("UPDATE ")
				updateStmt.WriteString(tableName)
				updateStmt.WriteString(" SET reserved_before=DATE_ADD_MILLIS(NOW_UTC(), ?)")
				updateStmt.WriteString(" WHERE tenant_id=? AND id IN (")
				for i, k := range foundKeys {
					if i > 0 {
						updateStmt.WriteString(",")
					}
					updateStmt.WriteString(strconv.Itoa(k.ID))
				}
				updateStmt.WriteString(")")
				_, err = tx.ExecContext(ctx, updateStmt.String(), durMillis, tenantID)
				if err != nil {
					tx.Rollback()
					return reservedKeys, errors.Trace(err)
				}
			}
			err = tx.Commit()
			if err != nil {
				return reservedKeys, errors.Trace(err)
			}
			reservedKeys = append(reservedKeys, foundKeys...)
		case "pgx", "sqlite":
			// PostgreSQL supports RETURNING
			var stmt strings.Builder
			stmt.WriteString("UPDATE ")
			stmt.WriteString(tableName)
			stmt.WriteString(" SET reserved_before=DATE_ADD_MILLIS(NOW_UTC(), ?)")
			stmt.WriteString(" WHERE tenant_id=? AND id IN (")
			writeIDList(&stmt)
			stmt.WriteString(")")
			if !forceful {
				stmt.WriteString(" AND reserved_before<=NOW_UTC()")
			}
			stmt.WriteString(" RETURNING id")
			rows, err := svc.db.QueryContext(ctx, stmt.String(), durMillis, tenantID)
			if err != nil {
				return reservedKeys, errors.Trace(err)
			}
			for rows.Next() {
				var key busstopapi.BusStopKey
				err = rows.Scan(&key)
				if err != nil {
					rows.Close()
					return reservedKeys, errors.Trace(err)
				}
				reservedKeys = append(reservedKeys, key)
			}
			rows.Close()
		case "mssql":
			// SQL Server supports OUTPUT INSERTED
			var stmt strings.Builder
			stmt.WriteString("UPDATE ")
			stmt.WriteString(tableName)
			stmt.WriteString(" SET reserved_before=DATE_ADD_MILLIS(NOW_UTC(), ?)")
			stmt.WriteString(" OUTPUT INSERTED.id")
			stmt.WriteString(" WHERE tenant_id=? AND id IN (")
			writeIDList(&stmt)
			stmt.WriteString(")")
			if !forceful {
				stmt.WriteString(" AND reserved_before<=NOW_UTC()")
			}
			rows, err := svc.db.QueryContext(ctx, stmt.String(), durMillis, tenantID)
			if err != nil {
				return reservedKeys, errors.Trace(err)
			}
			for rows.Next() {
				var key busstopapi.BusStopKey
				err = rows.Scan(&key)
				if err != nil {
					rows.Close()
					return reservedKeys, errors.Trace(err)
				}
				reservedKeys = append(reservedKeys, key)
			}
			rows.Close()
		}
	}
	return reservedKeys, nil
}
