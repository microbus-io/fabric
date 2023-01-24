# Package `examples.directory`

The `directory.example` microservice is an example of a microservice that provides a CRUD API backed by a SQL database.

## Adding SQL Support

It only takes a few steps to add SQL support to a microservice.

Step 1: Edit `service.yaml` to define a SQL database. In this example case we name a `mariadb` database `Maria`.

```yaml
databases:
  - name: Maria
    type: mariadb
```

Step 2: Run `go generate`.

Step 3: Create a SQL schema migration script `resources/maria/1.sql`. This script will automatically be executed when the microservice starts and connects to the database. A migration script is only executed once in the order of its file name, so create `2.sql` etc. if and when changes to the schema are required. File names may include arbitrary text after the first dot, like so `2.create-table.sql`.

```sql
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

Step 4: Use `svc.MariaDatabase()` to access the [sharded database](../structure/shardedsql.md) from any of the endpoints of the microservice. The name `MariaDatabase` is derived from the name you chose in step 1.

Step 5: Add a `Maria` config property in `config.yaml` pointing to the location of your database server. The name `Maria` is the one you chose in step 1.

```yaml
all:
  Maria: "root:secret1234@tcp(127.0.0.1:3306)/example_shard_%d"
```

## Connecting to the Database

If you don't have a `MariaDB` database running already, you can use `Docker` to install and run it locally:

```cmd
docker pull mariadb
docker run -p 3306:3306 --name mariadb1 -e MARIADB_ROOT_PASSWORD=secret1234 -d mariadb
```

The connection string to the database is pulled from `examples/main/config.yaml` by the configurator and served to the `directory.example` microservice. Uncomment and edit as necessary to point to the location of your `MariaDB` database.

```yaml
all:
  Maria: "root:secret1234@tcp(127.0.0.1:3306)/microbus_examples_shard%d"
```

Note that the `%d` at the end of the database name is important. It's used to denote the database shard number and will be filled with the number `1`. There is no need to `CREATE DATABASE microbus_examples_shard1`. It will be created automatically.

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
