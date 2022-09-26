package controllers

import (
	"fmt"
	"net/http"
	"time"
)

func IsEnvReady(prNumber int) bool {
	prEnvURL := fmt.Sprintf("http://ephenvtestpr%d.eastus.cloudapp.azure.com/", prNumber)
	client := http.Client{
		Timeout: 2 * time.Second,
	}
	resp, err := client.Get(prEnvURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}
