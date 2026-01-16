package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/sadewadee/google-scraper/internal/domain"
	"github.com/tealeg/xlsx/v3"
)

// BusinessListingService provides business logic for business listings
type BusinessListingService struct {
	repo domain.BusinessListingRepository
}

// NewBusinessListingService creates a new service
func NewBusinessListingService(repo domain.BusinessListingRepository) *BusinessListingService {
	return &BusinessListingService{repo: repo}
}

// List retrieves business listings with filters and pagination
func (s *BusinessListingService) List(ctx context.Context, filter domain.BusinessListingFilter) ([]*domain.BusinessListing, int, error) {
	return s.repo.List(ctx, filter)
}

// ListByJobID retrieves business listings for a specific job
func (s *BusinessListingService) ListByJobID(ctx context.Context, jobID string, limit, offset int) ([]*domain.BusinessListing, int, error) {
	return s.repo.ListByJobID(ctx, jobID, limit, offset)
}

// GetByID retrieves a single business listing by ID
func (s *BusinessListingService) GetByID(ctx context.Context, id int64) (*domain.BusinessListing, error) {
	return s.repo.GetByID(ctx, id)
}

// GetCategories returns distinct categories
func (s *BusinessListingService) GetCategories(ctx context.Context, limit int) ([]string, error) {
	return s.repo.GetCategories(ctx, limit)
}

// GetCities returns distinct cities
func (s *BusinessListingService) GetCities(ctx context.Context, limit int) ([]string, error) {
	return s.repo.GetCities(ctx, limit)
}

// Stats returns aggregate statistics
func (s *BusinessListingService) Stats(ctx context.Context) (*domain.BusinessListingStats, error) {
	return s.repo.Stats(ctx)
}

// CountByJobID counts business listings for a job
func (s *BusinessListingService) CountByJobID(ctx context.Context, jobID string) (int, error) {
	return s.repo.CountByJobID(ctx, jobID)
}

// AvailableColumns returns the list of available columns for export
func (s *BusinessListingService) AvailableColumns() []string {
	return []string{
		"title",
		"category",
		"address",
		"phone",
		"website",
		"email",
		"latitude",
		"longitude",
		"city",
		"country",
		"review_count",
		"review_rating",
		"status",
		"price_range",
		"link",
		"place_id",
		"cid",
	}
}

// ExportCSV exports business listings to CSV format
func (s *BusinessListingService) ExportCSV(ctx context.Context, w io.Writer, filter domain.BusinessListingFilter, columns []string) error {
	if len(columns) == 0 {
		columns = s.AvailableColumns()
	}

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write header
	if err := csvWriter.Write(columns); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}

	return s.repo.Stream(ctx, filter, func(listing *domain.BusinessListing) error {
		row := s.listingToRow(listing, columns)
		return csvWriter.Write(row)
	})
}

// ExportCSVByJobID exports business listings for a job to CSV format
func (s *BusinessListingService) ExportCSVByJobID(ctx context.Context, w io.Writer, jobID string, columns []string) error {
	if len(columns) == 0 {
		columns = s.AvailableColumns()
	}

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write header
	if err := csvWriter.Write(columns); err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}

	return s.repo.StreamByJobID(ctx, jobID, func(listing *domain.BusinessListing) error {
		row := s.listingToRow(listing, columns)
		return csvWriter.Write(row)
	})
}

// ExportJSON exports business listings to JSON format
func (s *BusinessListingService) ExportJSON(ctx context.Context, w io.Writer, filter domain.BusinessListingFilter) error {
	// Write opening bracket
	if _, err := w.Write([]byte("[\n")); err != nil {
		return err
	}

	first := true
	err := s.repo.Stream(ctx, filter, func(listing *domain.BusinessListing) error {
		if !first {
			if _, err := w.Write([]byte(",\n")); err != nil {
				return err
			}
		}
		first = false
		data, err := json.MarshalIndent(listing, "", "  ")
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	})
	if err != nil {
		return err
	}

	// Write closing bracket
	_, err = w.Write([]byte("\n]"))
	return err
}

// ExportJSONByJobID exports business listings for a job to JSON format
func (s *BusinessListingService) ExportJSONByJobID(ctx context.Context, w io.Writer, jobID string) error {
	// Write opening bracket
	if _, err := w.Write([]byte("[\n")); err != nil {
		return err
	}

	first := true
	err := s.repo.StreamByJobID(ctx, jobID, func(listing *domain.BusinessListing) error {
		if !first {
			if _, err := w.Write([]byte(",\n")); err != nil {
				return err
			}
		}
		first = false
		data, err := json.MarshalIndent(listing, "", "  ")
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	})
	if err != nil {
		return err
	}

	// Write closing bracket
	_, err = w.Write([]byte("\n]"))
	return err
}

// ExportXLSX exports business listings to XLSX format
func (s *BusinessListingService) ExportXLSX(ctx context.Context, w io.Writer, filter domain.BusinessListingFilter, columns []string) error {
	if len(columns) == 0 {
		columns = s.AvailableColumns()
	}

	wb := xlsx.NewFile()
	sheet, err := wb.AddSheet("Business Listings")
	if err != nil {
		return fmt.Errorf("create xlsx sheet: %w", err)
	}

	// Write header
	headerRow := sheet.AddRow()
	for _, col := range columns {
		cell := headerRow.AddCell()
		cell.SetString(col)
	}

	// Stream data
	err = s.repo.Stream(ctx, filter, func(listing *domain.BusinessListing) error {
		row := sheet.AddRow()
		values := s.listingToRow(listing, columns)
		for _, val := range values {
			cell := row.AddCell()
			cell.SetString(val)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return wb.Write(w)
}

// ExportXLSXByJobID exports business listings for a job to XLSX format
func (s *BusinessListingService) ExportXLSXByJobID(ctx context.Context, w io.Writer, jobID string, columns []string) error {
	if len(columns) == 0 {
		columns = s.AvailableColumns()
	}

	wb := xlsx.NewFile()
	sheet, err := wb.AddSheet("Business Listings")
	if err != nil {
		return fmt.Errorf("create xlsx sheet: %w", err)
	}

	// Write header
	headerRow := sheet.AddRow()
	for _, col := range columns {
		cell := headerRow.AddCell()
		cell.SetString(col)
	}

	// Stream data
	err = s.repo.StreamByJobID(ctx, jobID, func(listing *domain.BusinessListing) error {
		row := sheet.AddRow()
		values := s.listingToRow(listing, columns)
		for _, val := range values {
			cell := row.AddCell()
			cell.SetString(val)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return wb.Write(w)
}

// listingToRow converts a business listing to a row based on selected columns
func (s *BusinessListingService) listingToRow(listing *domain.BusinessListing, columns []string) []string {
	row := make([]string, len(columns))
	for i, col := range columns {
		row[i] = s.getColumnValue(listing, col)
	}
	return row
}

// getColumnValue extracts a column value from a business listing
func (s *BusinessListingService) getColumnValue(listing *domain.BusinessListing, column string) string {
	switch column {
	case "title":
		return listing.Title
	case "category":
		if listing.Category != nil {
			return *listing.Category
		}
	case "address":
		if listing.Address != nil {
			return *listing.Address
		}
	case "phone":
		if listing.Phone != nil {
			return *listing.Phone
		}
	case "website":
		if listing.Website != nil {
			return *listing.Website
		}
	case "email":
		if len(listing.Emails) > 0 {
			return strings.Join(listing.Emails, ", ")
		}
	case "latitude":
		if listing.Latitude != nil {
			return fmt.Sprintf("%f", *listing.Latitude)
		}
	case "longitude":
		if listing.Longitude != nil {
			return fmt.Sprintf("%f", *listing.Longitude)
		}
	case "city":
		if listing.AddressCity != nil {
			return *listing.AddressCity
		}
	case "country":
		if listing.AddressCountry != nil {
			return *listing.AddressCountry
		}
	case "review_count":
		return fmt.Sprintf("%d", listing.ReviewCount)
	case "review_rating":
		if listing.ReviewRating != nil {
			return fmt.Sprintf("%.1f", *listing.ReviewRating)
		}
	case "status":
		if listing.Status != nil {
			return *listing.Status
		}
	case "price_range":
		if listing.PriceRange != nil {
			return *listing.PriceRange
		}
	case "link":
		if listing.Link != nil {
			return *listing.Link
		}
	case "place_id":
		if listing.PlaceID != nil {
			return *listing.PlaceID
		}
	case "cid":
		if listing.CID != nil {
			return *listing.CID
		}
	}
	return ""
}
