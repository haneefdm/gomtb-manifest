package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func GetUrlContent(fileURL string) ([]byte, error) {
	// Configure the HTTP proxy (replace with your proxy details)
	var client *http.Client = nil
	if ProxyUrl != "" {
		parsedProxyURL, err := url.Parse(ProxyUrl)
		if err != nil {
			return nil, fmt.Errorf("Error parsing proxy URL: %v", err)
		}

		// Create a custom HTTP client with proxy settings
		client = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(parsedProxyURL),
			},
		}
	}
	if client == nil {
		client = &http.Client{}
	}

	// Make the GET request
	resp, err := client.Get(fileURL)
	if err != nil {
		return nil, fmt.Errorf("Error making GET request: %v", err)
	}
	defer resp.Body.Close()

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d %s", resp.StatusCode, resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Error reading response body: %v", err)
	}
	return bodyBytes, nil
}
