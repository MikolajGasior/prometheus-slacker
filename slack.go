package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

const ColorInfo = "#00BFFF"    // Deep Sky Blue
const ColorSuccess = "#00FF00" // Lime
const ColorWarn = "#FFD700"    // Gold
const ColorError = "#DC143C"   // Crimson

type SlackMessage struct {
	Blocks       []SlackBlock `json:"blocks"`
	DetailBlocks []SlackBlock `json:"detail_blocks,omitempty"`
	ActionBlocks []SlackBlock `json:"action_blocks,omitempty"`
}

type SlackBlock struct {
	Type      string               `json:"type"`
	Text      *SlackBlockText      `json:"text,omitempty"`
	Accessory *SlackBlockAccessory `json:"accessory,omitempty"`
	Elements  []SlackBlockElement  `json:"elements,omitempty"`
}

type SlackBlockText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type SlackBlockAccessory struct {
	Type     string                `json:"type"`
	ImageURL string                `json:"image_url,omitempty"`
	AltText  string                `json:"alt_text,omitempty"`
	Text     SlackBlockElementText `json:"text"`
	URL      string                `json:"url,omitempty"`
	ActionID string                `json:"action_id,omitempty"`
	Value    string                `json:"value,omitempty"`
}

type SlackBlockElement struct {
	Type  string                `json:"type"`
	Text  SlackBlockElementText `json:"text"`
	Value string                `json:"value"`
	URL   string                `json:"url"`
}

type SlackBlockElementText struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Emoji bool   `json:"emoji"`
}

type SlackWebhook struct {
	Url         string          `json:"url"`
	ShowDetails map[string]bool `json:"show_details"`
	ShowActions map[string]bool `json:"show_actions"`
}

func (n *SlackWebhook) SendMessage(msg SlackMessage) error {
	log.Print("Sending Slack message...")

	payload, err := json.Marshal(msg)
	if err != nil {
		return errors.New("Failed to marshal Slack message: " + err.Error())
	}
	res, err := http.Post(n.Url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return errors.New("Failed to send Slack message - got error: " + err.Error())
	}
	res.Body.Close()

	log.Print("Slack message sent")

	return nil
}
