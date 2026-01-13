package handlers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/sadewadee/google-scraper/gmaps"
)

// GlobalResultServiceInterface defines methods for global results access
type GlobalResultServiceInterface interface {
	ListAll(ctx context.Context, limit, offset int) ([][]byte, int, error)
}

// ResultHandler handles result-related HTTP requests (global view)
type ResultHandler struct {
	results GlobalResultServiceInterface
}

// NewResultHandler creates a new ResultHandler
func NewResultHandler(results GlobalResultServiceInterface) *ResultHandler {
	return &ResultHandler{
		results: results,
	}
}

// List handles GET /api/v2/results - returns all results globally
func (h *ResultHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	// Parse pagination
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}

	offset := (page - 1) * perPage

	results, total, err := h.results.ListAll(r.Context(), perPage, offset)
	if err != nil {
		RenderError(w, http.StatusInternalServerError, "Failed to get results: "+err.Error())
		return
	}

	// Parse JSON data
	var parsedResults []json.RawMessage
	for _, data := range results {
		parsedResults = append(parsedResults, json.RawMessage(data))
	}

	response := NewPaginatedResponse(parsedResults, total, page, perPage)
	RenderJSON(w, http.StatusOK, response)
}

// Download handles GET /api/v2/results/download - exports all results
func (h *ResultHandler) Download(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RenderError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}

	switch format {
	case "json":
		h.downloadJSON(w, r)
	case "csv":
		h.downloadCSV(w, r)
	case "xlsx":
		h.downloadXLSX(w, r)
	default:
		RenderError(w, http.StatusBadRequest, "Invalid format. Use 'json', 'csv', or 'xlsx'")
	}
}

func (h *ResultHandler) downloadJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=all-results.json")

	w.Write([]byte("["))
	first := true

	// Stream in batches
	offset := 0
	batchSize := 1000

	for {
		results, _, err := h.results.ListAll(r.Context(), batchSize, offset)
		if err != nil {
			break
		}

		if len(results) == 0 {
			break
		}

		for _, data := range results {
			if !first {
				w.Write([]byte(","))
			}
			first = false
			w.Write(data)
		}

		offset += batchSize
	}

	w.Write([]byte("]"))
}

func (h *ResultHandler) downloadCSV(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=all-results.csv")

	availableColumns := getGlobalAvailableColumns()
	selectedColumns := parseGlobalSelectedColumns(r.URL.Query().Get("columns"), availableColumns)

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write Header
	if err := writer.Write(selectedColumns); err != nil {
		return
	}

	// Stream in batches
	offset := 0
	batchSize := 1000

	for {
		results, _, err := h.results.ListAll(r.Context(), batchSize, offset)
		if err != nil {
			break
		}

		if len(results) == 0 {
			break
		}

		for _, data := range results {
			var entry gmaps.Entry
			if err := json.Unmarshal(data, &entry); err != nil {
				continue
			}

			record := make([]string, len(selectedColumns))
			for i, col := range selectedColumns {
				record[i] = availableColumns[col](&entry)
			}

			writer.Write(record)
		}

		offset += batchSize
		writer.Flush()
	}
}

func (h *ResultHandler) downloadXLSX(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=all-results.xlsx")

	availableColumns := getGlobalAvailableColumns()
	selectedColumns := parseGlobalSelectedColumns(r.URL.Query().Get("columns"), availableColumns)

	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Results"
	f.SetSheetName("Sheet1", sheetName)

	// Write header row
	for i, col := range selectedColumns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, col)
	}

	// Style the header
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0E0E0"}, Pattern: 1},
	})
	lastCol, _ := excelize.CoordinatesToCellName(len(selectedColumns), 1)
	f.SetCellStyle(sheetName, "A1", lastCol, headerStyle)

	rowNum := 2
	offset := 0
	batchSize := 1000

	for {
		results, _, err := h.results.ListAll(r.Context(), batchSize, offset)
		if err != nil {
			break
		}

		if len(results) == 0 {
			break
		}

		for _, data := range results {
			var entry gmaps.Entry
			if err := json.Unmarshal(data, &entry); err != nil {
				continue
			}

			for i, col := range selectedColumns {
				cell, _ := excelize.CoordinatesToCellName(i+1, rowNum)
				f.SetCellValue(sheetName, cell, availableColumns[col](&entry))
			}
			rowNum++
		}

		offset += batchSize
	}

	// Auto-fit column widths
	for i := range selectedColumns {
		colName, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheetName, colName, colName, 15)
	}

	if err := f.Write(w); err != nil {
		log.Printf("error writing XLSX to response: %v", err)
	}
}

// getGlobalAvailableColumns returns the map of available export columns
func getGlobalAvailableColumns() map[string]func(e *gmaps.Entry) string {
	return map[string]func(e *gmaps.Entry) string{
		"Title":           func(e *gmaps.Entry) string { return e.Title },
		"Address":         func(e *gmaps.Entry) string { return e.Address },
		"Phone":           func(e *gmaps.Entry) string { return e.Phone },
		"Website":         func(e *gmaps.Entry) string { return e.WebSite },
		"Category":        func(e *gmaps.Entry) string { return e.Category },
		"Rating":          func(e *gmaps.Entry) string { return fmt.Sprintf("%.1f", e.ReviewRating) },
		"Reviews":         func(e *gmaps.Entry) string { return fmt.Sprintf("%d", e.ReviewCount) },
		"Latitude":        func(e *gmaps.Entry) string { return fmt.Sprintf("%f", e.Latitude) },
		"Longitude":       func(e *gmaps.Entry) string { return fmt.Sprintf("%f", e.Longtitude) },
		"Place ID":        func(e *gmaps.Entry) string { return e.PlaceID },
		"Google Maps URL": func(e *gmaps.Entry) string { return e.Link },
		"Description":     func(e *gmaps.Entry) string { return e.Description },
		"Status":          func(e *gmaps.Entry) string { return e.Status },
		"Timezone":        func(e *gmaps.Entry) string { return e.Timezone },
		"Price Range":     func(e *gmaps.Entry) string { return e.PriceRange },
		"Data ID":         func(e *gmaps.Entry) string { return e.DataID },
		"Email":           func(e *gmaps.Entry) string { return strings.Join(e.Emails, ", ") },
		"Opening Hours": func(e *gmaps.Entry) string {
			var parts []string
			for day, hours := range e.OpenHours {
				parts = append(parts, fmt.Sprintf("%s: %s", day, strings.Join(hours, ", ")))
			}
			return strings.Join(parts, "; ")
		},
	}
}

// parseGlobalSelectedColumns parses and validates requested columns
func parseGlobalSelectedColumns(colsParam string, availableColumns map[string]func(e *gmaps.Entry) string) []string {
	var selectedColumns []string
	if colsParam != "" {
		requested := strings.Split(colsParam, ",")
		for _, col := range requested {
			col = strings.TrimSpace(col)
			if _, ok := availableColumns[col]; ok {
				selectedColumns = append(selectedColumns, col)
			}
		}
	}

	// Default columns if none selected or invalid
	if len(selectedColumns) == 0 {
		selectedColumns = []string{
			"Title", "Address", "Phone", "Website", "Category", "Rating", "Reviews",
			"Latitude", "Longitude", "Place ID", "Google Maps URL",
		}
	}
	return selectedColumns
}
