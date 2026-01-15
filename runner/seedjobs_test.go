package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateSeedJobsFromKeywords(t *testing.T) {
	tests := []struct {
		name        string
		cfg         SeedJobConfig
		expectedErr bool
		expectedLen int
	}{
		{
			name: "Valid keywords",
			cfg: SeedJobConfig{
				Keywords: []string{"pizza", "coffee"},
				LangCode: "en",
			},
			expectedErr: false,
			expectedLen: 2,
		},
		{
			name: "Empty keywords",
			cfg: SeedJobConfig{
				Keywords: []string{},
			},
			expectedErr: true,
			expectedLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobs, err := CreateSeedJobsFromKeywords(tt.cfg)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, jobs, tt.expectedLen)
			}
		})
	}
}

func TestFormatGeoCoordinates(t *testing.T) {
	tests := []struct {
		name     string
		lat      float64
		lon      float64
		expected string
	}{
		{
			name:     "Both zero",
			lat:      0,
			lon:      0,
			expected: "",
		},
		{
			name:     "Non-zero",
			lat:      40.7128,
			lon:      -74.0060,
			expected: "40.712800,-74.006000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := FormatGeoCoordinates(tt.lat, tt.lon)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
