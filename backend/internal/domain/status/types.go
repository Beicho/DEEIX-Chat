package status

import "time"

// UsageStat summarizes recent successful calls for one platform model.
type UsageStat struct {
	PlatformModelName string
	TotalCalls        int64
	SuccessCalls      int64
}

// ModelDetail is the application-facing availability view for one model.
type ModelDetail struct {
	ModelName    string
	Availability float64
	Status       string
	LastChecked  time.Time
}

// ModelsStatus is the aggregate public model status view.
type ModelsStatus struct {
	OverallStatus string
	LastUpdated   time.Time
	Models        []ModelDetail
}
