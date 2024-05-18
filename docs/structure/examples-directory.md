# Package `examples/directory`

The `directory.example` microservice is an example of a microservice that provides a CRUD API backed by a SQL database.
For the sake of this example, if a connection to the SQL database cannot be established, the microservice emulates a database in-memory.

## Adding SQL Support

It takes a few steps to add SQL support to a microservice.

Step 1: Edit `service.yaml` to define a configuration property to represent the connection string.

```yaml
configs:
  - signature: SQL() (dsn string)
    description: SQL is the connection string to the database.
```

Step 2: Run `go generate` to create the `svc.SQL()` method corresponding to the `SQL` configuration property.

```cmd
go generate
```

Step 3: Define the database connection `db *sql.DB` as a member property of the `Service`, open it in `OnStartup` and close it in `OnShutdown`.

```go
import _ "github.com/go-sql-driver/mysql"

type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE

	db *sql.DB
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	dsn := svc.SQL()
	if dsn != "" {
		svc.db, err = sql.Open("mysql", dsn)
		if err == nil {
			err = svc.db.PingContext(ctx)
		}
		if err != nil {
			return errors.Trace(err)
		}
	}
	return nil
}

// OnShutdown is called when the microservice is shut down.
func (svc *Service) OnShutdown(ctx context.Context) (err error) {
	if svc.db != nil {
		svc.db.Close()
		svc.db = nil
	}
	return nil
}
```

## Connecting to the Database

This example requires a MariaDB database instance. If you don't already have one installed, you can add it to Docker using:

```cmd
docker pull mariadb
docker run -p 3306:3306 --name mariadb-1 -e MARIADB_ROOT_PASSWORD=secret1234 -d mariadb
```

Next, create a database named `microbus_examples`.

<img src="examples-directory-1.png" width="498">
<p>

From the `Exec` panel of the `mariadb-1` container, type:

```cmd
mysql -uroot -psecret1234
```

And then use the SQL command prompt to create the database:

```sql
CREATE DATABASE microbus_examples;
```

The connection string to the database is pulled from `examples/main/config.yaml` by the configurator and served to the `directory.example` microservice. Adjust it as necessary to point to the location of your MariaDB database.

```yaml
directory.example:
  SQL: "root:secret1234@tcp(127.0.0.1:3306)/microbus_examples"
```

## Try It Out

To `Create` a new person in the directory:

http://localhost:8080/directory.example/create?person.firstName=Harry&person.lastName=Potter&person.email=harry.potter@hogwarts.edu.wiz

```json
{"created":{"birthday":null,"email":"harry.potter@hogwarts.edu.wiz","firstName":"Harry","key":{"seq":1},"lastName":"Potter"}}
```

To `Update` a record:

http://localhost:8080/directory.example/update?person.key.seq=1&person.firstName=Harry&person.lastName=Potter&person.email=harry.potter@hogwarts.edu.wiz&person.birthday=1980-07-31

```json
{"updated":{"birthday":"1980-07-31T00:00:00Z","email":"harry.potter@hogwarts.edu.wiz","firstName":"Harry","key":{"seq":1},"lastName":"Potter"},"ok":true}
```

To `List` all keys:

http://localhost:8080/directory.example/list

```json
{"keys":[{"seq":1}]}
```

To `Load` a record:

http://localhost:8080/directory.example/load?key.seq=1

```json
{"person":{"birthday":"1980-07-30T00:00:00Z","email":"harry.potter@hogwarts.edu.wiz","firstName":"Harry","key":{"seq":1},"lastName":"Potter"},"ok":true}
```

To `Delete` a record:

http://localhost:8080/directory.example/delete?key.seq=1

```json
{"ok":true}
```
