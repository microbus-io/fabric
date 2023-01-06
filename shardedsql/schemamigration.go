package shardedsql

// SchemaMigration represents a single SQL statement used to migrate the schema of the database.
// This is typically a CREATE TABLE or ALTER TABLE statement.
// The statements are guaranteed to run in order of the sequence number within the context of a
// globally unique name. Good practice is to use the name of the owner microservice.
// Names are limited to 256 ASCII characters.
type SchemaMigration struct {
	Name      string
	Sequence  int
	Statement string
}
