package calculator

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/microbus-io/fabric/connector"
	"github.com/microbus-io/fabric/errors"
)

// Service is a calculator microservice
type Service struct {
	*connector.Connector
}

// NewService creates a new calculator microservice
func NewService() *Service {
	s := &Service{
		Connector: connector.New("calculator.example"),
	}
	s.SetDescription("The Calculator microservice performs simple mathematical operations.")
	s.Subscribe("/arithmetic", s.Arithmetic)
	s.Subscribe("/square", s.Square)
	return s
}

// Arithmetic perform an arithmetic operation between two integers x and y given an operator op
func (s *Service) Arithmetic(w http.ResponseWriter, r *http.Request) error {
	// Read and parse query arguments
	x := r.URL.Query().Get("x")
	y := r.URL.Query().Get("y")
	op := r.URL.Query().Get("op")
	if op == " " {
		op = "+" // + is interpreted as a space in URLs
	}

	xx, err := strconv.ParseInt(x, 10, 32)
	if err != nil {
		return errors.Trace(err)
	}
	yy, _ := strconv.ParseInt(y, 10, 32)
	if err != nil {
		return errors.Trace(err)
	}
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
	default:
		return errors.Newf("invalid operator '%s'", op)
	}

	// Print the result
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(struct {
		X      int64  `json:"x"`
		Op     string `json:"op"`
		Y      int64  `json:"y"`
		Result int64  `json:"result"`
	}{
		xx,
		op,
		yy,
		rr,
	})
	return errors.Trace(err)
}

// Square prints the square of the integer x
func (s *Service) Square(w http.ResponseWriter, r *http.Request) error {
	// Read and parse query argument
	x := r.URL.Query().Get("x")
	xx, err := strconv.ParseInt(x, 10, 32)
	if err != nil {
		return errors.Trace(err)
	}

	// Print the result
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(struct {
		X      int64 `json:"x"`
		Result int64 `json:"result"`
	}{
		xx,
		xx * xx,
	})
	return errors.Trace(err)
}
