package cryptures

// Optional request fields are pointers so that "not set" can be told apart
// from a zero value. These helpers make literals convenient:
//
//	params := &cryptures.PortfolioParams{TokenTypes: "native,fungible", PageSize: cryptures.Int64(20)}

// String returns a pointer to v.
func String(v string) *string { return &v }

// Int64 returns a pointer to v.
func Int64(v int64) *int64 { return &v }

// Float64 returns a pointer to v.
func Float64(v float64) *float64 { return &v }

// Bool returns a pointer to v.
func Bool(v bool) *bool { return &v }
