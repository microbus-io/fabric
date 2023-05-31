# Package `shardedsql`

The `shardedsql` package is built on top of the standard library `sql` package and extends it to facilitate sharding a SQL database and differentially migrating its schema.

## Sharding

The core construct is the `DB` which represents a sharded SQL database. The `DB` includes a collection of `Shard`s that wrap the standard library's `sql.DB` and connect to one of the shard databases. The shard databases typically reside on different physical machines but can also be collocated on a single machine, as is the case during testing. Sharding is achieved by means of allocating sharding keys to a shard and later accessing the shard designated to that key. A common scenario is to use tenant IDs as the sharding keys resulting in a tenant's data all being on the same shard.

Opening a `DB` requires a data source name pattern that includes `%d` as a placeholder for the shard index. 

```go
// A different host for each shard database
db, err := shardedsql.Open(ctx, "mariadb", "username:password@tcp(db-shard%d.host:3306)/db")
// Same host but a different database for each shard database
db, err := shardedsql.Open(ctx, "mariadb", "username:password@tcp(db.host:3306)/db%d")
```

Initially a `DB` includes a single shard: shard number 1. Additional shards are added by registering them in the `microbus_shards` table on shard 1. The `id` column is the index of the shard and the `locked` column indicates whether the shard accepts allocation of new sharding keys or whether it is full.

```sql
CREATE TABLE IF NOT EXISTS microbus_shards (
	id INT NOT NULL,
	locked BOOL NOT NULL DEFAULT FALSE,
	PRIMARY KEY (id)
)
```

Warning: shards must not be changed while the application is running. Errors will occur if some of the microservices have recognized the change while others have not.

Sharding can be based on any data. Common practice is to shard by tenant so that the entirety of the tenant's data resides on one shard. `Allocate` is used to create a new sharding key and allocate it to an unlocked shard and `ShardOf` is used to obtain the shard designated to a sharding key.

```go
// Allocate a new sharding key for the tenant and insert it to its designated shard
shardingKey, err := db.Allocate()
shard := db.ShardOf(shardingKey)
res, err := shard.Exec("INSERT INTO tenants (tenant_id) VALUES (?)", shardingKey)
```

Allocations are stored in the `microbus_sharding_keys` table on shard 1. `AUTO_INCREMENT` guarantees the uniqueness of the keys.

```sql
CREATE TABLE IF NOT EXISTS microbus_sharding_keys (
	id BIGINT NOT NULL AUTO_INCREMENT,
	shard_id INT NOT NULL,
	PRIMARY KEY (id)
)
```

Shard 1 is critical to the operation of the system and is a single point of failure. High availability can be achieved with replication.

## Schema Migration

In a microservices architecture, microservices are upgraded and deployed in an unpredictable manner. If those microservices depend on a particular database schema, it is necessary to keep track of the current schema and upgrade it to the latest as needed. In `Microbus`, every microservice owns a piece of the database schema (often a single table). The microservice is required to provide schema migration scripts for any differential change it wants to introduce to the schema. This enables each of the shard to execute those scripts that have not yet been executed on that shard, bringing itself up to date.

Each shard keeps track of the migration scripts that have executed in the `microbus_schema_migrations` table. The `name` and `seq` columns identify the migration sequence order and order of execution. Migration scripts are guaranteed to execute in order of `seq` within the scope of a `name`. In the typical case, the host name of the microservice is used as the `name` of the migration sequence.

```sql
CREATE TABLE IF NOT EXISTS microbus_schema_migrations (
	name VARCHAR(256) CHARACTER SET ascii NOT NULL,
	seq INT NOT NULL,
	completed BOOL NOT NULL DEFAULT FALSE,
	completed_on DATETIME(3),
	locked_until DATETIME(3) NOT NULL DEFAULT UTC_TIMESTAMP(3),
	PRIMARY KEY (name, seq)
)
```

Migrations are executed when a microservice starts up, guaranteeing that the piece of the database that it relies on is up to date before it starts accepting requests.

## Testing Database

The `TestingDB` creates a randomly-named sharded database on localhost for development purposes. It is typically used in integration tests of microservices that depend on a database. `TestingDB` connects to the database on 127.0.0.1 on the default port (3306 for `MariaDB` and `MySQL`) using the default admin user ("root" for `MariaDB` and `MySQL`) with password `secret1234`. It creates 3 databases, one for each shard, using the pattern `testing_{hhmmss}_{random}_%d`. These databases are dropped when the database is closed.

```go
var testingDB TestingDB
testingDB.Open("mariadb")
defer testingDB.Close()
```
