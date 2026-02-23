package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

func UTCTime() string {
	now := time.Now()
	return now.UTC().Format("15:04") + " UTC"
}

func JSONCreate() {
	_, err := os.Open("warnings.json") // For read access.
	if err != nil {
		_, _ = os.Create("warnings.json")
	}
}

func fetchWarnings() {
	const url = "https://api.weather.gov/alerts/active?event=tornado%20warning,tornado%20watch,severe%20thunderstorm%20warning,severe%20thunderstorm%20watch,special%20weather%20statement"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Errorf("failed to fetch warnings: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		fmt.Errorf("API returned non-200 status: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Errorf("failed to read response body: %w", err)
	}

	fmt.Printf(string(body))
}
