package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type CodaAPIService struct {
	BaseURL  string
	APIToken string
	Timeout  time.Duration
	Client   *http.Client
}

func NewCodaAPIService(baseUrl string, apiToken string) *CodaAPIService {
	return &CodaAPIService{
		BaseURL:  baseUrl,
		APIToken: apiToken,
		Timeout:  30 * time.Second,
		Client:   &http.Client{Timeout: 30 * time.Second},
	}
}

type CodaCell struct {
	Column string      `json:"column"`
	Value  interface{} `json:"value"`
}

type CodaRow struct {
	Cells []CodaCell `json:"cells"`
}

type UpsertRowProductBacklogTableRequest struct {
	Rows []CodaRow `json:"rows"`
}

// CodaUpsertResponse represents the response from Coda's upsert API
type CodaUpsertResponse struct {
	RequestID   string   `json:"requestId"`
	AddedRowIDs []string `json:"addedRowIds"`
}

// CodaMutationStatus represents the status of a Coda mutation
type CodaMutationStatus struct {
	Completed bool   `json:"completed"`
	Warning   string `json:"warning,omitempty"`
}

func (c *CodaAPIService) UpsertRowIntoProductBacklogTable(documentName string) (*CodaUpsertResponse, error) {
	url := fmt.Sprintf("%s/docs/%s/tables/%s/rows", c.BaseURL, "izhKV9mlbM", "grid-Kf1wPPWSkE")

	// Debug: Log the URL and config
	log.Printf("Coda API Debug - URL: %s", url)
	log.Printf("Coda API Debug - Base URL: %s", c.BaseURL)
	log.Printf("Coda API Debug - Document Name: %s", documentName)

	// Create the request with the specified format
	req := UpsertRowProductBacklogTableRequest{
		Rows: []CodaRow{
			{
				Cells: []CodaCell{
					{
						Column: "c-lS_kSZIjjI",
						Value:  documentName,
					},
				},
			},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		log.Printf("Coda API Debug - JSON Marshal Error: %v", err)
		return nil, err
	}

	// Debug: Log the request payload
	log.Printf("Coda API Debug - Request Payload: %s", string(body))

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Coda API Debug - HTTP Request Creation Error: %v", err)
		return nil, err
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIToken))
	httpReq.Header.Set("Content-Type", "application/json")

	// Debug: Log request headers
	log.Printf("Coda API Debug - Request Headers: %v", httpReq.Header)

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		log.Printf("Coda API Debug - HTTP Request Error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Debug: Read response body for detailed error information
	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		log.Printf("Coda API Debug - Failed to read response body: %v", readErr)
	} else {
		log.Printf("Coda API Debug - Response Status: %d %s", resp.StatusCode, resp.Status)
		log.Printf("Coda API Debug - Response Body: %s", string(respBody))
		log.Printf("Coda API Debug - Response Headers: %v", resp.Header)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		if len(respBody) > 0 {
			return nil, fmt.Errorf("failed to upsert row: %s - Response: %s", resp.Status, string(respBody))
		}
		return nil, fmt.Errorf("failed to upsert row: %s", resp.Status)
	}

	// Parse the response to extract request ID
	var upsertResponse CodaUpsertResponse
	if err := json.Unmarshal(respBody, &upsertResponse); err != nil {
		log.Printf("Coda API Debug - Failed to parse response: %v", err)
		// Return success even if we can't parse the response
		return &CodaUpsertResponse{
			RequestID:   "unknown",
			AddedRowIDs: []string{},
		}, nil
	}

	log.Printf("Coda API Debug - Successfully upserted document: %s, Request ID: %s", documentName, upsertResponse.RequestID)
	return &upsertResponse, nil
}

// Helper method to create a more flexible upsert request
func (c *CodaAPIService) UpsertRowWithMultipleCells(docID, tableID string, cells map[string]interface{}, keyColumns []string) error {
	url := fmt.Sprintf("%s/docs/%s/tables/%s/rows", c.BaseURL, docID, tableID)

	// Convert map to CodaCell slice
	var codaCells []CodaCell
	for column, value := range cells {
		codaCells = append(codaCells, CodaCell{
			Column: column,
			Value:  value,
		})
	}

	req := UpsertRowProductBacklogTableRequest{
		Rows: []CodaRow{
			{
				Cells: codaCells,
			},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIToken))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("failed to upsert row: %s", resp.Status)
	}

	return nil
}

// CheckMutationStatus checks the status of a Coda mutation by request ID
func (c *CodaAPIService) CheckMutationStatus(requestID string) (*CodaMutationStatus, error) {
	url := fmt.Sprintf("%s/mutationStatus/%s", c.BaseURL, requestID)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIToken))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	log.Printf("Coda Mutation Status Debug - Response Status: %d", resp.StatusCode)
	log.Printf("Coda Mutation Status Debug - Response Body: %s", string(respBody))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("mutation status check failed: %s - Response: %s", resp.Status, string(respBody))
	}

	var mutationStatus CodaMutationStatus
	if err := json.Unmarshal(respBody, &mutationStatus); err != nil {
		return nil, fmt.Errorf("failed to parse mutation status response: %v", err)
	}

	return &mutationStatus, nil
}

// CodaRowResponse represents the response from Coda's get row API
type CodaRowResponse struct {
	ID     string                 `json:"id"`
	Type   string                 `json:"type"`
	Values map[string]interface{} `json:"values"`
}

// GetRowDevelopmentStatus fetches the development status from a Coda row
func (c *CodaAPIService) GetRowDevelopmentStatus(codaRowID string) (string, error) {
	url := fmt.Sprintf("%s/docs/%s/tables/%s/rows/%s", c.BaseURL, "izhKV9mlbM", "grid-Kf1wPPWSkE", codaRowID)

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIToken))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	log.Printf("Coda Row Debug - Response Status: %d", resp.StatusCode)
	log.Printf("Coda Row Debug - Response Body: %s", string(respBody))

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("get row failed: %s - Response: %s", resp.Status, string(respBody))
	}

	var rowResponse CodaRowResponse
	if err := json.Unmarshal(respBody, &rowResponse); err != nil {
		return "", fmt.Errorf("failed to parse row response: %v", err)
	}

	// Extract the development status from column c-VwYUU9Zhcy
	if devStatus, exists := rowResponse.Values["c-VwYUU9Zhcy"]; exists {
		if statusStr, ok := devStatus.(string); ok {
			return statusStr, nil
		}
		return fmt.Sprintf("%v", devStatus), nil // Convert to string if not already
	}

	return "", fmt.Errorf("development status column c-VwYUU9Zhcy not found in response")
}
