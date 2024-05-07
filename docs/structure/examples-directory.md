# Package `examples/directory`

The `directory.example` microservice is an example of a microservice that provides a CRUD API backed by a SQL database.

## Adding SQL Support

It only takes a few steps to add SQL support to a microservice.

Step 1: Create a database `microbus_examples` and a table `directory_persons` in it. For this example do this manually, but this is something you'll want to automate with schema migration tools.

```sql
CREATE DATABASE my_database

USE my_database

CREATE TABLE directory_persons (
	person_id BIGINT NOT NULL AUTO_INCREMENT,
	first_name VARCHAR(32) NOT NULL,
	last_name VARCHAR(32) NOT NULL,
	email_address VARCHAR(128) CHARACTER SET ascii NOT NULL,
	birthday DATE,
	PRIMARY KEY (person_id),
	CONSTRAINT UNIQUE INDEX (email_address)
) CHARACTER SET utf8
```

Step 2: Edit `service.yaml` to define a configuration property to represent the connection string.

```yaml
configs:
  - signature: SQL() (dsn string)
    description: SQL is the data source name to the MariaDB database. For example, root:secret@tcp(127.0.0.1:3306)/microbus_examples
```

Step 3: Run `go generate`.

Step 4: Define the database connection `db` as a member property of the `Service`, open it in `OnStartup` and close it in `OnShutdown`.

```go
type Service struct {
	*intermediate.Intermediate // DO NOT REMOVE

	db *sql.DB
}

// OnStartup is called when the microservice is started up.
func (svc *Service) OnStartup(ctx context.Context) (err error) {
	dsn := svc.SQL()
	if dsn != "" {
		svc.db, err = sql.Open("mysql", dsn)
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

If you don't have a MariaDB database running already, you can use `Docker` to install and run it locally:

```cmd
docker pull mariadb
docker run -p 3306:3306 --name mariadb1 -e MARIADB_ROOT_PASSWORD=secret1234 -d mariadb
```

The connection string to the database is pulled from `examples/main/config.yaml` by the configurator and served to the `directory.example` microservice. Uncomment and edit as necessary to point to the location of your MariaDB database.

```yaml
all:
  Maria: "root:secret1234@tcp(127.0.0.1:3306)/microbus_examples"
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
