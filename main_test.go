package main

import (
	"math"
	"strings"
	"testing"
)

func TestGenerateMeme(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		lat      float64
		lon      float64
		wantMeme bool
	}{
		{
			name:     "No filters",
			query:    "",
			lat:      0,
			lon:      0,
			wantMeme: true,
		},
		{
			name:     "Query filter - should match",
			query:    "code",
			lat:      0,
			lon:      0,
			wantMeme: true,
		},
		{
			name:     "Query filter - should not match",
			query:    "nonexistent",
			lat:      0,
			lon:      0,
			wantMeme: false,
		},
		{
			name:     "Location filter - near New York",
			query:    "",
			lat:      40.7128,
			lon:      -74.0060,
			wantMeme: true,
		},
		{
			name:     "Location and query filter - near London with 'coding'",
			query:    "coding",
			lat:      51.5074,
			lon:      -0.1278,
			wantMeme: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meme := generateMeme(tt.query, tt.lat, tt.lon)

			// Check if we got a meme when we wanted one
			if tt.wantMeme && meme.Title == "" {
				t.Errorf("generateMeme() wanted a meme but got empty response")
				return
			}

			// Check if we got an empty meme when we didn't want one
			if !tt.wantMeme && meme.Title != "" {
				t.Errorf("generateMeme() wanted empty response but got meme: %v", meme.Title)
				return
			}

			// Only check query matching if we expect a meme and have a query
			if tt.wantMeme && tt.query != "" && !strings.Contains(strings.ToLower(meme.Title), strings.ToLower(tt.query)) {
				t.Errorf("generateMeme() title %q doesn't contain query %q", meme.Title, tt.query)
			}

			// Only check location matching when coordinates are provided and we expect a meme
			if tt.wantMeme && tt.lat != 0 && tt.lon != 0 {
				found := false
				for _, m := range []struct{ lat, lon float64 }{
					{40.7128, -74.0060},  // New York
					{51.5074, -0.1278},   // London
					{35.6762, 139.6503},  // Tokyo
					{-33.8688, 151.2093}, // Sydney
				} {
					if calculateDistance(tt.lat, tt.lon, m.lat, m.lon) <= 5000.0 {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("generateMeme() location (%.4f, %.4f) not within range of any meme locations", tt.lat, tt.lon)
				}
			}
		})
	}
}

func TestCalculateDistance(t *testing.T) {
	tests := []struct {
		name    string
		lat1    float64
		lon1    float64
		lat2    float64
		lon2    float64
		want    float64
		epsilon float64
	}{
		{
			name:    "New York to London",
			lat1:    40.7128,
			lon1:    -74.0060,
			lat2:    51.5074,
			lon2:    -0.1278,
			want:    5570.0, // approximate distance in km
			epsilon: 1.0,    // allow 1km difference due to floating point precision
		},
		{
			name:    "Same location",
			lat1:    40.7128,
			lon1:    -74.0060,
			lat2:    40.7128,
			lon2:    -74.0060,
			want:    0.0,
			epsilon: 0.1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateDistance(tt.lat1, tt.lon1, tt.lat2, tt.lon2)
			if math.Abs(got-tt.want) > tt.epsilon {
				t.Errorf("calculateDistance() = %v, want %v (±%v)", got, tt.want, tt.epsilon)
			}
		})
	}
}

func TestGenerateMemeWithNoMatches(t *testing.T) {
	// Test case 1: Non-matching query
	meme := generateMeme("nonexistentquery", 0, 0)
	if meme.ID != 0 || meme.Title != "" || meme.URL != "" {
		t.Errorf("Expected empty meme for non-matching query, got: %+v", meme)
	}
	if meme.Query != "nonexistentquery" {
		t.Errorf("Expected query to be 'nonexistentquery', got: %s", meme.Query)
	}

	// Test case 2: Non-matching location (Antarctica)
	meme = generateMeme("", -82.8628, -135.0000) // Deep in Antarctica, far from all city locations
	if meme.ID != 0 || meme.Title != "" || meme.URL != "" {
		t.Errorf("Expected empty meme for non-matching location, got: %+v", meme)
	}
	if meme.Lat != -82.8628 || meme.Lon != -135.0000 {
		t.Errorf("Expected location (-82.8628,-135.0000), got: (%f,%f)", meme.Lat, meme.Lon)
	}
}
