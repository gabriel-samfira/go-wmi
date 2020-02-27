package wmi

import "errors"

// ErrNotFound is returned when the query yielded no results
var ErrNotFound = errors.New("Query returned empty set")
