package main

import (
	"encoding/json"
	"log"
)

type Config struct {
	Version            string                  `json:"version"`
	Port               string                  `json:"port"`
	SlackWebhooks      map[string]SlackWebhook `json:"slack_webhooks"`
	PrometheusURL      string                  `json:"prometheus_url"`
	ScrapperMinutes    int                     `json:"scrapper_minutes"`
	NotificationLevels []NotificationLevel     `json:"notification_levels"`
	Metrics            map[string]Metric       `json:"metrics"`
}

func (c *Config) SetFromJSON(b []byte) {
	err := json.Unmarshal(b, c)
	if err != nil {
		log.Fatal("Error setting config from JSON:", err.Error())
	}
}

type NotificationLevel struct {
	Color           string            `json:"color"`
	SlackWebhooks   []string          `json:"slack_webhooks"`
	SlackMessage    SlackMessage      `json:"slack_message"`
	LeverageMetrics map[string]string `json:"leverage_metrics"`
}

type Metric struct {
	DisplayName       string `json:"display_name"`
	Query             string `json:"query"`
	Threshold         string `json:"threshold"`
	LastValue         string `json:"last_value"`
	Leverage          bool   `json:"leverage,omitempty"`
	ThresholdExceeded bool   `json:"threshold_exceeded,omitempty"`
}
