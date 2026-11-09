-- DRIVER: mysql mariadb
CREATE TABLE persons (
	id BIGINT NOT NULL AUTO_INCREMENT,
	first_name VARCHAR(64) NOT NULL,
	last_name VARCHAR(64) NOT NULL,
	email VARCHAR(256) NOT NULL,
	birthday DATETIME(3) NULL,
	created_at DATETIME(3) NOT NULL,
	updated_at DATETIME(3) NOT NULL,

	CONSTRAINT persons_pk PRIMARY KEY (id),
	UNIQUE INDEX persons_idx_email (email)
);

-- DRIVER: pgx cockroachdb
CREATE TABLE persons (
	id BIGSERIAL,
	first_name VARCHAR(64) NOT NULL,
	last_name VARCHAR(64) NOT NULL,
	email VARCHAR(256) NOT NULL,
	birthday TIMESTAMP(3) NULL,
	created_at TIMESTAMP(3) NOT NULL,
	updated_at TIMESTAMP(3) NOT NULL,

	CONSTRAINT persons_pk PRIMARY KEY (id)
);
-- DRIVER: pgx cockroachdb
CREATE UNIQUE INDEX persons_idx_email ON persons (email);

-- DRIVER: mssql
CREATE TABLE persons (
	id BIGINT IDENTITY(1, 1),
	first_name NVARCHAR(64) NOT NULL,
	last_name NVARCHAR(64) NOT NULL,
	email NVARCHAR(256) NOT NULL,
	birthday DATETIME2(3) NULL,
	created_at DATETIME2(3) NOT NULL,
	updated_at DATETIME2(3) NOT NULL,

	CONSTRAINT persons_pk PRIMARY KEY (id),
	CONSTRAINT persons_idx_email UNIQUE (email)
);

-- DRIVER: sqlite
CREATE TABLE persons (
	id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
	first_name TEXT NOT NULL,
	last_name TEXT NOT NULL,
	email TEXT NOT NULL,
	birthday DATETIME,
	created_at DATETIME NOT NULL,
	updated_at DATETIME NOT NULL
);
-- DRIVER: sqlite
CREATE UNIQUE INDEX persons_idx_email ON persons (email);
