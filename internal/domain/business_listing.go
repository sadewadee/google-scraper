package domain

import "github.com/google/uuid"

// BusinessListing represents a normalized business listing
type BusinessListing struct {
	ID              int64       `json:"id"`
	ResultID        int64       `json:"result_id"`
	JobID           *string     `json:"job_id,omitempty"`
	PlaceID         *string     `json:"place_id,omitempty"`
	CID             *string     `json:"cid,omitempty"`
	Title           string      `json:"title"`
	Category        *string     `json:"category,omitempty"`
	Categories      []string    `json:"categories,omitempty"`
	Address         *string     `json:"address,omitempty"`
	Phone           *string     `json:"phone,omitempty"`
	Website         *string     `json:"website,omitempty"`
	Latitude        *float64    `json:"latitude,omitempty"`
	Longitude       *float64    `json:"longitude,omitempty"`
	AddressCity     *string     `json:"address_city,omitempty"`
	AddressCountry  *string     `json:"address_country,omitempty"`
	ReviewCount     int         `json:"review_count"`
	ReviewRating    *float64    `json:"review_rating,omitempty"`
	Status          *string     `json:"status,omitempty"`
	PriceRange      *string     `json:"price_range,omitempty"`
	Link            *string     `json:"link,omitempty"`
	CreatedAt       string      `json:"created_at"`
	Emails          []string    `json:"emails,omitempty"`
	EmailsWithInfo  []EmailInfo `json:"emails_with_info,omitempty"`
	ValidEmailCount int         `json:"valid_email_count"`
	TotalEmailCount int         `json:"total_email_count"`
}

// EmailInfo contains email with validation status
type EmailInfo struct {
	Email        string   `json:"email"`
	Status       string   `json:"status"`
	Score        *float64 `json:"score,omitempty"`
	IsAcceptable *bool    `json:"is_acceptable,omitempty"`
}

// BusinessListingFilter contains filter parameters for queries
type BusinessListingFilter struct {
	JobID       *uuid.UUID
	Search      string // Search in title, address, phone, category
	Category    string
	City        string
	Country     string
	MinRating   *float64
	HasEmail    *bool
	EmailStatus string // api_valid, api_invalid, pending, local_valid
	Page        int
	PerPage     int
	SortBy      string // created_at, review_rating, review_count, title
	SortOrder   string // asc, desc
}

// BusinessListingStats contains aggregate statistics
type BusinessListingStats struct {
	TotalListings int      `json:"total_listings"`
	TotalJobs     int      `json:"total_jobs"`
	TotalEmails   int      `json:"total_emails"`
	ValidEmails   int      `json:"valid_emails"`
	WithPhone     int      `json:"with_phone"`
	WithWebsite   int      `json:"with_website"`
	AvgRating     *float64 `json:"avg_rating,omitempty"`
}
