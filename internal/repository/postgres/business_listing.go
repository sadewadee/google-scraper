package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/google/uuid"
)

// BusinessListing represents a normalized business listing
type BusinessListing struct {
	ID              int64    `json:"id"`
	ResultID        int64    `json:"result_id"`
	JobID           *string  `json:"job_id,omitempty"`
	PlaceID         *string  `json:"place_id,omitempty"`
	CID             *string  `json:"cid,omitempty"`
	Title           string   `json:"title"`
	Category        *string  `json:"category,omitempty"`
	Categories      []string `json:"categories,omitempty"`
	Address         *string  `json:"address,omitempty"`
	Phone           *string  `json:"phone,omitempty"`
	Website         *string  `json:"website,omitempty"`
	Latitude        *float64 `json:"latitude,omitempty"`
	Longitude       *float64 `json:"longitude,omitempty"`
	AddressCity     *string  `json:"address_city,omitempty"`
	AddressCountry  *string  `json:"address_country,omitempty"`
	ReviewCount     int      `json:"review_count"`
	ReviewRating    *float64 `json:"review_rating,omitempty"`
	Status          *string  `json:"status,omitempty"`
	PriceRange      *string  `json:"price_range,omitempty"`
	Link            *string  `json:"link,omitempty"`
	CreatedAt       string   `json:"created_at"`
	Emails          []string `json:"emails,omitempty"`
	EmailsWithInfo  []EmailInfo `json:"emails_with_info,omitempty"`
	ValidEmailCount int      `json:"valid_email_count"`
	TotalEmailCount int      `json:"total_email_count"`
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

// BusinessListingRepository provides access to business_listings
type BusinessListingRepository struct {
	db *sql.DB
}

// NewBusinessListingRepository creates a new repository
func NewBusinessListingRepository(db *sql.DB) *BusinessListingRepository {
	return &BusinessListingRepository{db: db}
}

// List retrieves business listings with filters
func (r *BusinessListingRepository) List(ctx context.Context, filter BusinessListingFilter) ([]BusinessListing, int, error) {
	// Set defaults
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PerPage < 1 || filter.PerPage > 100 {
		filter.PerPage = 25
	}
	if filter.SortBy == "" {
		filter.SortBy = "created_at"
	}
	if filter.SortOrder == "" {
		filter.SortOrder = "desc"
	}

	// Build WHERE clause
	var conditions []string
	var args []interface{}
	argNum := 1

	if filter.JobID != nil {
		conditions = append(conditions, fmt.Sprintf("bl.job_id = $%d", argNum))
		args = append(args, filter.JobID.String())
		argNum++
	}

	if filter.Search != "" {
		// Use trigram similarity for fuzzy search
		searchPattern := "%" + filter.Search + "%"
		conditions = append(conditions, fmt.Sprintf(
			"(bl.title ILIKE $%d OR bl.address ILIKE $%d OR bl.phone ILIKE $%d OR bl.category ILIKE $%d)",
			argNum, argNum, argNum, argNum,
		))
		args = append(args, searchPattern)
		argNum++
	}

	if filter.Category != "" {
		conditions = append(conditions, fmt.Sprintf("bl.category = $%d", argNum))
		args = append(args, filter.Category)
		argNum++
	}

	if filter.City != "" {
		conditions = append(conditions, fmt.Sprintf("bl.address_city = $%d", argNum))
		args = append(args, filter.City)
		argNum++
	}

	if filter.Country != "" {
		conditions = append(conditions, fmt.Sprintf("bl.address_country = $%d", argNum))
		args = append(args, filter.Country)
		argNum++
	}

	if filter.MinRating != nil {
		conditions = append(conditions, fmt.Sprintf("bl.review_rating >= $%d", argNum))
		args = append(args, *filter.MinRating)
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Validate sort column
	validSortColumns := map[string]string{
		"created_at":    "bl.created_at",
		"review_rating": "bl.review_rating",
		"review_count":  "bl.review_count",
		"title":         "bl.title",
	}
	sortColumn, ok := validSortColumns[filter.SortBy]
	if !ok {
		sortColumn = "bl.created_at"
	}

	sortOrder := "DESC"
	if strings.ToLower(filter.SortOrder) == "asc" {
		sortOrder = "ASC"
	}

	// Count total
	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT bl.id)
		FROM business_listings bl
		%s
	`, whereClause)

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count query failed: %w", err)
	}

	// Main query with email aggregation
	offset := (filter.Page - 1) * filter.PerPage
	query := fmt.Sprintf(`
		SELECT
			bl.id, bl.result_id, bl.job_id, bl.place_id, bl.cid,
			bl.title, bl.category, bl.categories, bl.address, bl.phone,
			bl.website, bl.latitude, bl.longitude, bl.address_city, bl.address_country,
			bl.review_count, bl.review_rating, bl.status, bl.price_range, bl.link,
			bl.created_at,
			COALESCE(
				jsonb_agg(
					DISTINCT jsonb_build_object(
						'email', e.email,
						'status', e.validation_status,
						'score', e.api_score,
						'is_acceptable', e.is_acceptable
					)
				) FILTER (WHERE e.id IS NOT NULL),
				'[]'::jsonb
			) AS emails_info,
			COALESCE(array_agg(DISTINCT e.email) FILTER (WHERE e.id IS NOT NULL), ARRAY[]::TEXT[]) AS emails,
			COUNT(DISTINCT e.id) FILTER (WHERE e.is_acceptable = true) AS valid_email_count,
			COUNT(DISTINCT e.id) AS total_email_count
		FROM business_listings bl
		LEFT JOIN business_emails be ON be.business_listing_id = bl.id
		LEFT JOIN emails e ON e.id = be.email_id
		%s
		GROUP BY bl.id
		ORDER BY %s %s NULLS LAST
		LIMIT $%d OFFSET $%d
	`, whereClause, sortColumn, sortOrder, argNum, argNum+1)

	args = append(args, filter.PerPage, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list query failed: %w", err)
	}
	defer rows.Close()

	var listings []BusinessListing
	for rows.Next() {
		var bl BusinessListing
		var jobID, placeID, cid, category, address, phone, website sql.NullString
		var addressCity, addressCountry, status, priceRange, link sql.NullString
		var latitude, longitude, reviewRating sql.NullFloat64
		var categories []byte
		var emailsInfoJSON []byte
		var emailsArray []byte

		err := rows.Scan(
			&bl.ID, &bl.ResultID, &jobID, &placeID, &cid,
			&bl.Title, &category, &categories, &address, &phone,
			&website, &latitude, &longitude, &addressCity, &addressCountry,
			&bl.ReviewCount, &reviewRating, &status, &priceRange, &link,
			&bl.CreatedAt,
			&emailsInfoJSON, &emailsArray,
			&bl.ValidEmailCount, &bl.TotalEmailCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan failed: %w", err)
		}

		// Handle nullable fields
		if jobID.Valid {
			bl.JobID = &jobID.String
		}
		if placeID.Valid {
			bl.PlaceID = &placeID.String
		}
		if cid.Valid {
			bl.CID = &cid.String
		}
		if category.Valid {
			bl.Category = &category.String
		}
		if address.Valid {
			bl.Address = &address.String
		}
		if phone.Valid {
			bl.Phone = &phone.String
		}
		if website.Valid {
			bl.Website = &website.String
		}
		if latitude.Valid {
			bl.Latitude = &latitude.Float64
		}
		if longitude.Valid {
			bl.Longitude = &longitude.Float64
		}
		if addressCity.Valid {
			bl.AddressCity = &addressCity.String
		}
		if addressCountry.Valid {
			bl.AddressCountry = &addressCountry.String
		}
		if reviewRating.Valid {
			bl.ReviewRating = &reviewRating.Float64
		}
		if status.Valid {
			bl.Status = &status.String
		}
		if priceRange.Valid {
			bl.PriceRange = &priceRange.String
		}
		if link.Valid {
			bl.Link = &link.String
		}

		// Parse categories array
		if len(categories) > 0 {
			// PostgreSQL array format: {value1,value2}
			if err := json.Unmarshal(categories, &bl.Categories); err != nil {
				log.Printf("[BusinessListingRepository] Warning: failed to unmarshal categories for listing %d: %v", bl.ID, err)
			}
		}

		// Parse emails info JSON
		if len(emailsInfoJSON) > 0 {
			if err := json.Unmarshal(emailsInfoJSON, &bl.EmailsWithInfo); err != nil {
				log.Printf("[BusinessListingRepository] Warning: failed to unmarshal emails_info for listing %d: %v", bl.ID, err)
			}
		}

		// Parse emails array
		if len(emailsArray) > 0 {
			if err := json.Unmarshal(emailsArray, &bl.Emails); err != nil {
				log.Printf("[BusinessListingRepository] Warning: failed to unmarshal emails for listing %d: %v", bl.ID, err)
			}
		}

		listings = append(listings, bl)
	}

	return listings, total, nil
}

// GetByID retrieves a single business listing by ID
func (r *BusinessListingRepository) GetByID(ctx context.Context, id int64) (*BusinessListing, error) {
	query := `
		SELECT
			bl.id, bl.result_id, bl.job_id, bl.place_id, bl.cid,
			bl.title, bl.category, bl.categories, bl.address, bl.phone,
			bl.website, bl.latitude, bl.longitude, bl.address_city, bl.address_country,
			bl.review_count, bl.review_rating, bl.status, bl.price_range, bl.link,
			bl.created_at,
			COALESCE(
				jsonb_agg(
					DISTINCT jsonb_build_object(
						'email', e.email,
						'status', e.validation_status,
						'score', e.api_score,
						'is_acceptable', e.is_acceptable
					)
				) FILTER (WHERE e.id IS NOT NULL),
				'[]'::jsonb
			) AS emails_info,
			COALESCE(array_agg(DISTINCT e.email) FILTER (WHERE e.id IS NOT NULL), ARRAY[]::TEXT[]) AS emails,
			COUNT(DISTINCT e.id) FILTER (WHERE e.is_acceptable = true) AS valid_email_count,
			COUNT(DISTINCT e.id) AS total_email_count
		FROM business_listings bl
		LEFT JOIN business_emails be ON be.business_listing_id = bl.id
		LEFT JOIN emails e ON e.id = be.email_id
		WHERE bl.id = $1
		GROUP BY bl.id
	`

	var bl BusinessListing
	var jobID, placeID, cid, category, address, phone, website sql.NullString
	var addressCity, addressCountry, status, priceRange, link sql.NullString
	var latitude, longitude, reviewRating sql.NullFloat64
	var categories []byte
	var emailsInfoJSON []byte
	var emailsArray []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&bl.ID, &bl.ResultID, &jobID, &placeID, &cid,
		&bl.Title, &category, &categories, &address, &phone,
		&website, &latitude, &longitude, &addressCity, &addressCountry,
		&bl.ReviewCount, &reviewRating, &status, &priceRange, &link,
		&bl.CreatedAt,
		&emailsInfoJSON, &emailsArray,
		&bl.ValidEmailCount, &bl.TotalEmailCount,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get by id failed: %w", err)
	}

	// Handle nullable fields (same as List)
	if jobID.Valid {
		bl.JobID = &jobID.String
	}
	if placeID.Valid {
		bl.PlaceID = &placeID.String
	}
	if cid.Valid {
		bl.CID = &cid.String
	}
	if category.Valid {
		bl.Category = &category.String
	}
	if address.Valid {
		bl.Address = &address.String
	}
	if phone.Valid {
		bl.Phone = &phone.String
	}
	if website.Valid {
		bl.Website = &website.String
	}
	if latitude.Valid {
		bl.Latitude = &latitude.Float64
	}
	if longitude.Valid {
		bl.Longitude = &longitude.Float64
	}
	if addressCity.Valid {
		bl.AddressCity = &addressCity.String
	}
	if addressCountry.Valid {
		bl.AddressCountry = &addressCountry.String
	}
	if reviewRating.Valid {
		bl.ReviewRating = &reviewRating.Float64
	}
	if status.Valid {
		bl.Status = &status.String
	}
	if priceRange.Valid {
		bl.PriceRange = &priceRange.String
	}
	if link.Valid {
		bl.Link = &link.String
	}

	if len(categories) > 0 {
		json.Unmarshal(categories, &bl.Categories)
	}
	if len(emailsInfoJSON) > 0 {
		json.Unmarshal(emailsInfoJSON, &bl.EmailsWithInfo)
	}
	if len(emailsArray) > 0 {
		json.Unmarshal(emailsArray, &bl.Emails)
	}

	return &bl, nil
}

// GetCategories returns distinct categories
func (r *BusinessListingRepository) GetCategories(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT category, COUNT(*) as cnt
		FROM business_listings
		WHERE category IS NOT NULL AND category != ''
		GROUP BY category
		ORDER BY cnt DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("get categories failed: %w", err)
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			return nil, fmt.Errorf("scan category failed: %w", err)
		}
		categories = append(categories, category)
	}

	return categories, nil
}

// GetCities returns distinct cities
func (r *BusinessListingRepository) GetCities(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	query := `
		SELECT address_city, COUNT(*) as cnt
		FROM business_listings
		WHERE address_city IS NOT NULL AND address_city != ''
		GROUP BY address_city
		ORDER BY cnt DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("get cities failed: %w", err)
	}
	defer rows.Close()

	var cities []string
	for rows.Next() {
		var city string
		var count int
		if err := rows.Scan(&city, &count); err != nil {
			return nil, fmt.Errorf("scan city failed: %w", err)
		}
		cities = append(cities, city)
	}

	return cities, nil
}

// Stats returns aggregate statistics
func (r *BusinessListingRepository) Stats(ctx context.Context) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total_listings,
			COUNT(DISTINCT bl.job_id) as total_jobs,
			COUNT(DISTINCT e.id) as total_emails,
			COUNT(DISTINCT e.id) FILTER (WHERE e.is_acceptable = true) as valid_emails,
			AVG(bl.review_rating) FILTER (WHERE bl.review_rating IS NOT NULL) as avg_rating,
			COUNT(*) FILTER (WHERE bl.phone IS NOT NULL AND bl.phone != '') as with_phone,
			COUNT(*) FILTER (WHERE bl.website IS NOT NULL AND bl.website != '') as with_website
		FROM business_listings bl
		LEFT JOIN business_emails be ON be.business_listing_id = bl.id
		LEFT JOIN emails e ON e.id = be.email_id
	`

	var totalListings, totalJobs, totalEmails, validEmails, withPhone, withWebsite int
	var avgRating sql.NullFloat64

	err := r.db.QueryRowContext(ctx, query).Scan(
		&totalListings, &totalJobs, &totalEmails, &validEmails,
		&avgRating, &withPhone, &withWebsite,
	)
	if err != nil {
		return nil, fmt.Errorf("stats query failed: %w", err)
	}

	stats := map[string]interface{}{
		"total_listings": totalListings,
		"total_jobs":     totalJobs,
		"total_emails":   totalEmails,
		"valid_emails":   validEmails,
		"with_phone":     withPhone,
		"with_website":   withWebsite,
	}

	if avgRating.Valid {
		stats["avg_rating"] = avgRating.Float64
	} else {
		stats["avg_rating"] = nil
	}

	return stats, nil
}
