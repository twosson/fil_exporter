package collector

import "errors"

// ErrNoData indicates the collector found no data to collect, but had no other error.
var ErrNoData = errors.New("collector returned no data")

// IsNoDataError asserts that error is an error without data
func IsNoDataError(err error) bool {
	return err == ErrNoData
}
