package main

type GrafanaAlert struct {
	Alerts []Alert `json:"alerts"`
}

type Alert struct {
	Annotations  map[string]string  `json:"annotations"`
	DashboardURL string             `json:"dashboardURL"`
	EndsAt       string             `json:"endsAt"`
	Fingerprint  string             `json:"fingerprint"`
	GeneratorURL string             `json:"generatorURL"`
	Labels       map[string]string  `json:"labels"`
	PanelURL     string             `json:"panelURL"`
	SilenceURL   string             `json:"silenceURL"`
	StartsAt     string             `json:"startsAt"`
	Status       string             `json:"status"`
	ValueString  string             `json:"valueString"`
	Values       map[string]float64 `json:"values"`
}

type Message struct {
	ChatId string `json:"chat_id"`
	Text   string `json:"text"`
}
