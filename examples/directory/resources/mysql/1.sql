CREATE TABLE directory_persons (
	person_id BIGINT NOT NULL AUTO_INCREMENT,
	first_name VARCHAR(32) NOT NULL,
	last_name VARCHAR(32) NOT NULL,
	email_address VARCHAR(128) CHARACTER SET ascii NOT NULL,
	birthday DATE,
	PRIMARY KEY (person_id),
	CONSTRAINT UNIQUE INDEX (email_address)
) CHARACTER SET utf8
