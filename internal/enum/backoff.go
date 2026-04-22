package enum

import "slices"

type BackoffType string

type backoffEnum struct {
	Domain BackoffType
	Page   BackoffType
}

var Backoff backoffEnum = backoffEnum{
	Domain: "Domain",
	Page:   "Page",
}

func (p BackoffType) IsValid() bool {
	valid := []BackoffType{
		Backoff.Domain,
		Backoff.Page,
	}

	return slices.Contains(valid, p)
}
