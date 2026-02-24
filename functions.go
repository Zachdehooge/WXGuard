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

func JSONCheck() {
	_, err := os.Open("warnings.json")
	if err != nil {
		_, _ = os.Create("warnings.json")
	}

	_, err = os.Open("config.json")
	if err != nil {
		_, _ = os.Create("config.json")
	}

}

func fetchWarningToJson() {
	const url = "https://api.weather.gov/alerts/active?event=tornado%20warning,tornado%20watch,severe%20thunderstorm%20warning,severe%20thunderstorm%20watch,special%20weather%20statement"

	resp, err := http.Get(url)
	if err != nil {
		_ = fmt.Errorf("failed to fetch warnings: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_ = fmt.Errorf("API returned non-200 status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = fmt.Errorf("failed to read response body: %w", err)
	}

	os.WriteFile("warnings.json", []byte(body), 0666)
}

func addChannel(torchannel string, svrstormchannel string, winterchannel string, swschannel string) {
	data := []byte(fmt.Sprintf(
		`{"tornado":{"channel":"%s"},"svrstorm":{"channel":"%s"},"winter":{"channel":"%s"},"sws":{"channel":"%s"}}`,
		torchannel, svrstormchannel, winterchannel, swschannel,
	))
	os.WriteFile("config.json", data, 0666)
}

func testAlert() {

}
