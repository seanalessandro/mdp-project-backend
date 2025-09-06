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
	Rows       []CodaRow `json:"rows"`
	KeyColumns []string  `json:"keyColumns"`
}

func (c *CodaAPIService) UpsertRowIntoProductBacklogTable(documentName string) error {
	url := fmt.Sprintf("%s/docs/%s/tables/%s/rows", c.BaseURL, "74D8gDLFK1", "grid-Kf1wPPWSkE")

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
		KeyColumns: []string{"c-lS_kSZIjjI"},
	}

	body, err := json.Marshal(req)
	if err != nil {
		log.Printf("Coda API Debug - JSON Marshal Error: %v", err)
		return err
	}

	// Debug: Log the request payload
	log.Printf("Coda API Debug - Request Payload: %s", string(body))

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Printf("Coda API Debug - HTTP Request Creation Error: %v", err)
		return err
	}

	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIToken))
	httpReq.Header.Set("Content-Type", "application/json")

	// Debug: Log request headers
	log.Printf("Coda API Debug - Request Headers: %v", httpReq.Header)

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		log.Printf("Coda API Debug - HTTP Request Error: %v", err)
		return err
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
			return fmt.Errorf("failed to upsert row: %s - Response: %s", resp.Status, string(respBody))
		}
		return fmt.Errorf("failed to upsert row: %s", resp.Status)
	}

	log.Printf("Coda API Debug - Successfully upserted document: %s", documentName)
	return nil
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
		KeyColumns: keyColumns,
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
