package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"time"

	"github.com/SunilKividor/bajaj-finserv-health-assignment/models"
)

const (
	initialURL = "https://bfhldevapigw.healthrx.co.in/hiring/generateWebhook"
	maxRetries = 4
)

func main() {
	log.Println("Application starting...")

	myName := "Sunil Kumar"
	myRegNo := "RA2211050010049"
	myEmail := "sk3870@srmist.edu.in"

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	initialReqData := models.InitialRequest{
		Name:  myName,
		RegNo: myRegNo,
		Email: myEmail,
	}

	initialResp, err := sendInitialRequest(client, initialURL, initialReqData)
	if err != nil {
		log.Fatalf("Error: Failed to get data: %v", err)
	}

	users := initialResp.Data.UserData.Users
	log.Printf("Received %d users in data.", len(users))

	outcome := solveMutualFollowers(users)
	log.Printf("Mutual followers length: %v", len(outcome))
	log.Printf("Mutual followers identified: %v", outcome)

	resultPayload := models.ResultPayload{
		RegNo:   myRegNo,
		Outcome: outcome,
	}

	err = sendResultWithRetry(client, initialResp.Webhook, initialResp.AccessToken, resultPayload)
	if err != nil {
		log.Fatalf("Error: Failed to send result to webhook after %d retries: %v", maxRetries, err)
	}

	log.Println("Application done.")
}

func sendInitialRequest(client *http.Client, url string, payload models.InitialRequest) (*models.InitialResponse, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal initial request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create initial request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute initial request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read initial response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("initial request failed with status %s", resp.Status)
	}

	var initialResponse models.InitialResponse
	err = json.Unmarshal(bodyBytes, &initialResponse)
	if err != nil {
		log.Printf("Error: Failed to unmarshal JSON, Response body")
		return nil, fmt.Errorf("failed to unmarshal initial response JSON: %w", err)
	}

	log.Printf("body body body: %v", initialResponse)

	if initialResponse.Webhook == "" || initialResponse.AccessToken == "" {
		return nil, fmt.Errorf("missing webhook or accessToken")
	}

	if initialResponse.Data.UserData.Users == nil {
		return nil, fmt.Errorf("data.users.users array is null or missing")
	}

	return &initialResponse, nil
}

func solveMutualFollowers(users []models.User) [][]int {
	followingMap := make(map[int]map[int]bool)
	for _, user := range users {
		if _, ok := followingMap[user.ID]; !ok {
			followingMap[user.ID] = make(map[int]bool)
		}
		for _, followedID := range user.Follows {
			followingMap[user.ID][followedID] = true
		}
	}

	mutualPairs := make([][]int, 0)
	addedPairs := make(map[string]bool)

	for followerID, followedMap := range followingMap {
		for followedID := range followedMap {
			if otherFollowsMap, ok := followingMap[followedID]; ok {
				if _, followsBack := otherFollowsMap[followerID]; followsBack {
					pair := []int{followerID, followedID}
					sort.Ints(pair)
					pairKey := fmt.Sprintf("%d-%d", pair[0], pair[1])

					if !addedPairs[pairKey] {
						mutualPairs = append(mutualPairs, pair)
						addedPairs[pairKey] = true
					}
				}
			}
		}
	}
	return mutualPairs
}

func sendResultWithRetry(client *http.Client, webhookURL, token string, payload models.ResultPayload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal result payload: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		log.Printf("Attempt %d", attempt+1)
		req, err := http.NewRequest("POST", webhookURL, bytes.NewBuffer(jsonData))
		if err != nil {
			lastErr = fmt.Errorf("failed to create result request: %v", err.Error())
			time.Sleep(1 * time.Second)
			continue
		}

		req.Header.Set("Authorization", token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to execute result request : %v", err.Error())
			log.Printf("WARN: %v", lastErr)
			time.Sleep(1 * time.Second)
			continue
		}

		bodyBytes, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		if readErr != nil {
			log.Printf("Failed to read response body from webhook: %v", readErr.Error())
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Printf("Attempt %d: Successfully sent result (Status: %s). Response: %s", attempt+1, resp.Status, string(bodyBytes))
			return nil
		}

		lastErr = fmt.Errorf("attempt %d: result request failed with status %s: %s", attempt+1, resp.Status, string(bodyBytes))
		log.Println(lastErr.Error())
		if attempt < maxRetries-1 {
			time.Sleep(time.Duration(attempt+1) * time.Second)
		}
	}

	return fmt.Errorf("failed after %d attempts. Last error: %v", maxRetries, lastErr)
}
