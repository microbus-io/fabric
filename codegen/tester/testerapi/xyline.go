package testerapi

// XYLine is a non-primitive type with a nested non-primitive type.
type XYLine struct {
	Start XYCoord `json:"start,omitempty"`
	End   XYCoord `json:"end,omitempty"`
}
