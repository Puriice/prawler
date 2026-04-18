package enum

import "slices"

type PageStatus string

type pageStatusEnum struct {
	Pending PageStatus
	Parsed  PageStatus
	Indexed PageStatus
	Failed  PageStatus
	Skipped PageStatus
}

var Page pageStatusEnum = pageStatusEnum{
	Pending: "Pending",
	Parsed:  "Parsed",
	Indexed: "Indexed",
	Failed:  "Failed",
	Skipped: "Skipped",
}

func (p PageStatus) IsValid() bool {
	valid := []PageStatus{
		Page.Pending,
		Page.Parsed,
		Page.Indexed,
		Page.Failed,
		Page.Skipped,
	}

	return slices.Contains(valid, p)
}
