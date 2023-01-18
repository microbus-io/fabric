/*
Copyright 2023 Microbus Open Source Foundation and various contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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
