package examples
package main

// Example: Custom Slack notification plugin

import (























}	return err	_, err := http.Post(s.webhookURL, "application/json", bytes.NewReader(data))	data, _ := json.Marshal(payload)	payload := map[string]string{"text": message}func (s *SlackNotifier) Send(message string) error {}	return "slack"func (s *SlackNotifier) Name() string {}	return &SlackNotifier{webhookURL: webhookURL}func NewSlackNotifier(webhookURL string) *SlackNotifier {}	webhookURL stringtype SlackNotifier struct {)	"net/http"	"encoding/json"	"bytes"