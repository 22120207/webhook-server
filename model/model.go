package model

import "time"

type GrafanaAlert struct {
	Alerts []Alert `json:"alerts"`
}

type Alert struct {
	Annotations  map[string]string  `json:"annotations"`
	DashboardURL string             `json:"dashboardURL"`
	EndsAt       time.Time          `json:"endsAt"`
	Fingerprint  string             `json:"fingerprint"`
	GeneratorURL string             `json:"generatorURL"`
	Labels       map[string]string  `json:"labels"`
	PanelURL     string             `json:"panelURL"`
	SilenceURL   string             `json:"silenceURL"`
	StartsAt     time.Time          `json:"startsAt"`
	Status       string             `json:"status"`
	ValueString  string             `json:"valueString"`
	Values       map[string]float64 `json:"values"`
}

type TelegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

type DiscordMessage struct {
	Content   string `json:"content"`
	AvatarURL string `json:"avatar_url,omitempty"`
}
