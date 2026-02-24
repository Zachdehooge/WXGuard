package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type NWSAlerts struct {
	Features []AlertFeature `json:"features"`
}

type AlertFeature struct {
	ID         string          `json:"id"`
	Properties AlertProperties `json:"properties"`
}

type AlertProperties struct {
	Event       string `json:"event"`
	Headline    string `json:"headline"`
	Description string `json:"description"`
	AreaDesc    string `json:"areaDesc"`
	Severity    string `json:"severity"`
	Urgency     string `json:"urgency"`
	Certainty   string `json:"certainty"`
	SenderName  string `json:"senderName"`
	Effective   string `json:"effective"`
	Expires     string `json:"expires"`
}

type ChannelConfig struct {
	Tornado  ChannelInfo `json:"tornado"`
	Svrstorm ChannelInfo `json:"svrstorm"`
	Winter   ChannelInfo `json:"winter"`
	Sws      ChannelInfo `json:"sws"`
}

type ChannelInfo struct {
	Channel string `json:"channel"`
}

type SentAlerts struct {
	SentIDs []string `json:"sent_ids"`
}

var globalSession *discordgo.Session

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

	_, err = os.Open("sent_alerts.json")
	if err != nil {
		_, _ = os.Create("sent_alerts.json")
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
	config := ChannelConfig{
		Tornado:  ChannelInfo{Channel: torchannel},
		Svrstorm: ChannelInfo{Channel: svrstormchannel},
		Winter:   ChannelInfo{Channel: winterchannel},
		Sws:      ChannelInfo{Channel: swschannel},
	}
	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile("config.json", data, 0666)
}

func getConfig() ChannelConfig {
	data, err := os.ReadFile("config.json")
	if err != nil {
		return ChannelConfig{}
	}
	var config ChannelConfig
	_ = json.Unmarshal(data, &config)
	return config
}

func getSentAlerts() SentAlerts {
	data, err := os.ReadFile("sent_alerts.json")
	if err != nil {
		return SentAlerts{SentIDs: []string{}}
	}
	var sent SentAlerts
	_ = json.Unmarshal(data, &sent)
	return sent
}

func saveSentAlerts(sent SentAlerts) {
	data, _ := json.MarshalIndent(sent, "", "  ")
	os.WriteFile("sent_alerts.json", data, 0666)
}

func classifyAlert(event string) string {
	event = strings.ToLower(event)

	if strings.Contains(event, "tornado") {
		return "tornado"
	}
	if strings.Contains(event, "severe thunderstorm") {
		return "svrstorm"
	}
	if strings.Contains(event, "winter") ||
		strings.Contains(event, "blizzard") ||
		strings.Contains(event, "ice storm") ||
		strings.Contains(event, "snow") ||
		strings.Contains(event, "freezing") ||
		strings.Contains(event, "wind chill") ||
		strings.Contains(event, "frost") ||
		strings.Contains(event, "cold") {
		return "winter"
	}
	if strings.Contains(event, "special weather") {
		return "sws"
	}

	return ""
}

func getChannelID(alertType string) string {
	config := getConfig()

	switch alertType {
	case "tornado":
		return config.Tornado.Channel
	case "svrstorm":
		return config.Svrstorm.Channel
	case "winter":
		return config.Winter.Channel
	case "sws":
		return config.Sws.Channel
	}

	return ""
}

func sendAlertToDiscord(alert AlertFeature) {
	alertType := classifyAlert(alert.Properties.Event)
	if alertType == "" {
		return
	}

	channelID := getChannelID(alertType)
	if channelID == "" {
		return
	}

	channelID = strings.Trim(channelID, "<>")

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s - %s", alert.Properties.Event, alert.Properties.Severity),
		Description: alert.Properties.Headline,
		Color:       getSeverityColor(alert.Properties.Severity),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Area",
				Value:  alert.Properties.AreaDesc,
				Inline: false,
			},
			{
				Name:   "Timing",
				Value:  fmt.Sprintf("Effective: %s\nExpires: %s", formatTime(alert.Properties.Effective), formatTime(alert.Properties.Expires)),
				Inline: true,
			},
			{
				Name:   "Urgency",
				Value:  alert.Properties.Urgency,
				Inline: true,
			},
			{
				Name:   "Source",
				Value:  alert.Properties.SenderName,
				Inline: true,
			},
		},
	}

	if len(alert.Properties.Description) > 0 {
		desc := alert.Properties.Description
		if len(desc) > 1000 {
			desc = desc[:1000] + "..."
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Details",
			Value:  desc,
			Inline: false,
		})
	}

	_, err := globalSession.ChannelMessageSendEmbed(channelID, embed)
	if err != nil {
		fmt.Printf("Failed to send alert to channel %s: %v\n", channelID, err)
	}
}

func getSeverityColor(severity string) int {
	severity = strings.ToLower(severity)
	switch severity {
	case "extreme":
		return 0xFF0000
	case "severe":
		return 0xFF8C00
	case "moderate":
		return 0xFFA500
	case "minor":
		return 0xFFFF00
	default:
		return 0x0099FF
	}
}

func formatTime(timeStr string) string {
	if timeStr == "" {
		return "N/A"
	}
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return timeStr
	}
	return t.Format("01/02/2006 3:04 PM")
}

func CheckAndSendAlerts() {
	const url = "https://api.weather.gov/alerts/active?event=tornado%20warning,tornado%20watch,severe%20thunderstorm%20warning,severe%20thunderstorm%20watch,special%20weather%20statement"

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Failed to fetch warnings: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("API returned non-200 status: %d\n", resp.StatusCode)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read response body: %v\n", err)
		return
	}

	var alerts NWSAlerts
	if err := json.Unmarshal(body, &alerts); err != nil {
		fmt.Printf("Failed to parse alerts: %v\n", err)
		return
	}

	sentAlerts := getSentAlerts()
	sentIDs := make(map[string]bool)
	for _, id := range sentAlerts.SentIDs {
		sentIDs[id] = true
	}

	currentActiveIDs := make(map[string]bool)
	for _, alert := range alerts.Features {
		currentActiveIDs[alert.ID] = true
	}

	prunedIDs := []string{}
	for _, id := range sentAlerts.SentIDs {
		if currentActiveIDs[id] {
			prunedIDs = append(prunedIDs, id)
		}
	}

	newSentIDs := prunedIDs

	for _, alert := range alerts.Features {
		if !sentIDs[alert.ID] {
			sendAlertToDiscord(alert)
			newSentIDs = append(newSentIDs, alert.ID)
		}
	}

	saveSentAlerts(SentAlerts{SentIDs: newSentIDs})
}

func testAlert() {

}
