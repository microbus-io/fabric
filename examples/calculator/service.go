package calculator

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/microbus-io/fabric/connector"
)

// Service is a calculator microservice
type Service struct {
	*connector.Connector
}

// NewService creates a new calculator microservice
func NewService() *Service {
	s := &Service{
		Connector: connector.NewConnector(),
	}
	s.SetHostName("calculator.example")
	s.Subscribe(443, "/arithmetic", s.Arithmetic)
	s.Subscribe(443, "/square", s.Square)
	return s
}

// Arithmetic perform an arithmetic operation between two integers x and y given an
// operator op
func (s *Service) Arithmetic(w http.ResponseWriter, r *http.Request) {
	// Read and parse query arguments
	x := r.URL.Query().Get("x")
	y := r.URL.Query().Get("y")
	op := r.URL.Query().Get("op")
	if op == " " {
		op = "+" // + is interpreted as a space in URLs
	}

	xx, _ := strconv.ParseInt(x, 10, 32)
	yy, _ := strconv.ParseInt(y, 10, 32)
	var rr int64

	// Perform the arithmetic operation
	switch op {
	case "*":
		rr = xx * yy
	case "+":
		rr = xx + yy
	case "-":
		rr = xx - yy
	case "/":
		rr = xx / yy
	}

	// Print the result
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("%d %s %d = %d", xx, op, yy, rr)))
}

// Square prints the square of the integer x
func (s *Service) Square(w http.ResponseWriter, r *http.Request) {
	// Read and parse query argument
	x := r.URL.Query().Get("x")
	xx, _ := strconv.ParseInt(x, 10, 32)

	// Print the result
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(fmt.Sprintf("%d ^ 2 = %d", xx, xx*xx)))
}
