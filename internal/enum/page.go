package enum

type PageStatus string

type pageStatusEnum struct {
	Pending PageStatus
	Parsed  PageStatus
	Indexed PageStatus
	Failted PageStatus
	Skipped PageStatus
}

var Page pageStatusEnum = pageStatusEnum{
	Pending: "Pending",
	Parsed:  "Parsed",
	Indexed: "Indexed",
	Failted: "Failed",
	Skipped: "Skipped",
}
