package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
)

type webHookData struct {
	IPs  []string
	User string
	Hash string
}

func (app *booksingApp) fireWebHook(d webHookData) {

	jsonData, err := json.Marshal(d)
	if err != nil {
		slog.Error("Failed to parse JSON", "error", err)
		return
	}

	request, err := http.NewRequest("POST", app.cfg.WebHookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		slog.Error("Failed to make new request", "error", err)
		return
	}
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		slog.Error("Failed to make request", "error", err)
	}
	defer response.Body.Close()
}
