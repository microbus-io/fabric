/*
Copyright (c) 2023 Microbus LLC and various contributors

This file and the project encapsulating it are the confidential intellectual property of Microbus LLC.
Neither may be used, copied or distributed without the express written consent of Microbus LLC.
*/

package shardedsql

import "sort"

// StatementSequence is a sequence of SQL statements that must be executed in order.
// It is used to migrate the schema of the database and typically includes a CREATE TABLE or ALTER TABLE statement.
type StatementSequence struct {
	Name       string
	Statements map[int]string
}

// NewStatementSequence creates a new sequence of SQL statements.
func NewStatementSequence(name string) *StatementSequence {
	return &StatementSequence{
		Name:       name,
		Statements: map[int]string{},
	}
}

// Insert adds a statement to the sequence.
func (ss *StatementSequence) Insert(sequenceNumber int, statement string) {
	if ss.Statements == nil {
		ss.Statements = map[int]string{}
	}
	ss.Statements[sequenceNumber] = statement
}

// Order returns the sequence numbers of all statements, in order.
func (ss *StatementSequence) Order() (sequenceNumbers []int) {
	sequenceNumbers = make([]int, 0, len(ss.Statements))
	for k := range ss.Statements {
		sequenceNumbers = append(sequenceNumbers, k)
	}
	sort.Slice(sequenceNumbers, func(i, j int) bool {
		return sequenceNumbers[i] < sequenceNumbers[j]
	})
	return sequenceNumbers
}
