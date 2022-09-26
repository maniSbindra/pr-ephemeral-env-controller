package controllers

import (
	"net/http"
	"time"
)

func IsEnvReady(ephEnvUrl string) bool {
	client := http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Get(ephEnvUrl)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}
