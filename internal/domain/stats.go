package domain

// Stats contains dashboard statistics
type Stats struct {
	Jobs    JobStats    `json:"jobs"`
	Workers WorkerStats `json:"workers"`
	Places  PlaceStats  `json:"places"`
}

// JobStats contains job-related statistics
type JobStats struct {
	Total     int `json:"total"`
	Pending   int `json:"pending"`
	Queued    int `json:"queued"`
	Running   int `json:"running"`
	Paused    int `json:"paused"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Cancelled int `json:"cancelled"`
}

// PlaceStats contains place-related statistics
type PlaceStats struct {
	TotalScraped int `json:"total_scraped"`
	Today        int `json:"today"`
	TotalEmails  int `json:"total_emails"`
	RatePerHour  int `json:"rate_per_hour"`
}
