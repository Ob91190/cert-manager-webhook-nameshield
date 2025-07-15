package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"time"
)

const (
	NameShieldLiveDnsTestBaseUrl = "https://ote-api.nameshield.net/dns/v2/"
)

const (
	NameShieldLiveDnsBaseUrl = "https://api.nameshield.net/dns/v2/"
)

type NameShieldClient struct {
	apiKey              string
	dumpRequestResponse bool
}

type NameShieldRecord struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Data       string `json:"data"`
	TTL        int    `json:"ttl,omitempty"`
	Comment    string `json:"comment,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	ModifiedAt string `json:"modified_at,omitempty"`
	ModifiedBy string `json:"modified_by,omitempty"`
}

type NameShieldRecordUpdate struct {
	Data    string `json:"data"`
	Comment string `json:"comment,omitempty"`
}

type NameShieldRecordSearchResults struct {
	Message string `json:"message"`
	Data    struct {
		Total   int                   `json:"total"`
		Limit   int                   `json:"limit"`
		Offset  int                   `json:"offset"`
		Results []NameShieldRecord    `json:"results"`
	} `json:"data"`
}

func NewNameShieldClient(apiKey string) *NameShieldClient {
	return &NameShieldClient{
		apiKey:              apiKey,
		dumpRequestResponse: false,
	}
}

func (c *NameShieldClient) nameShieldRecordsUrl(domain string) string {
	return fmt.Sprintf("%s/zones/%s/records", NameShieldLiveDnsBaseUrl, domain)
}

func (c *NameShieldClient) doRequest(req *http.Request, readResponseBody bool) (int, []byte, error) {
	if c.dumpRequestResponse {
		dump, _ := httputil.DumpRequest(req, true)
		fmt.Printf("Request: %q\n", dump)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	client := http.Client{
		Timeout: 30 * time.Second,
	}

	res, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}

	if c.dumpRequestResponse {
		dump, _ := httputil.DumpResponse(res, true)
		fmt.Printf("Response: %q\n", dump)
	}

	if res.StatusCode == http.StatusOK && readResponseBody {
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return 0, nil, err
		}
		return res.StatusCode, data, nil
	}

	return res.StatusCode, nil, nil
}

func (c *NameShieldClient) HasTxtRecord(domain *string, name *string) (bool, error) {
	// API NameShield: GET /dns/v2/zones/{zonename}/records?name={name}&type=TXT
	url := fmt.Sprintf("%s?name=%s&type=TXT", c.nameShieldRecordsUrl(*domain), *name)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}

	status, responseBody, err := c.doRequest(req, true)
	if err != nil {
		return false, err
	}

	if status == http.StatusNotFound {
		return false, nil
	} else if status == http.StatusOK {
		// Parse response body to check if any records were found
		var searchResults NameShieldRecordSearchResults
		if err := json.Unmarshal(responseBody, &searchResults); err != nil {
			return false, fmt.Errorf("failed to parse response: %v", err)
		}
		
		// Check if we found any TXT records with the specified name
		return searchResults.Data.Total > 0, nil
	} else {
		return false, fmt.Errorf("unexpected HTTP status: %d, response: %s", status, string(responseBody))
	}
}

func (c *NameShieldClient) CreateTxtRecord(domain *string, name *string, value *string, ttl int) error {
	// API NameShield: POST /dns/v2/zones/{zonename}/records
	// Body: {"name": "", "type": "TXT", "data": "value", "comment": "optional"}
	record := NameShieldRecord{
		Name:    *name,
		Type:    "TXT",
		Data:    *value,
		Comment: "Created by cert-manager",
	}
	
	body, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("cannot marshall to json: %v", err)
	}

	url := c.nameShieldRecordsUrl(*domain)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	status, responseBody, err := c.doRequest(req, true)
	if err != nil {
		return err
	}

	if status != http.StatusCreated && status != http.StatusOK {
		return fmt.Errorf("failed creating TXT record: %d, %s", status, string(responseBody))
	}

	return nil
}

func (c *NameShieldClient) UpdateTxtRecord(domain *string, name *string, value *string, ttl int) error {
	// API NameShield: PUT /dns/v2/zones/{zonename}/records/{id}
	// Body: {"data": "new_value", "comment": "optional"}
	recordUpdate := NameShieldRecordUpdate{
		Data:    *value,
		Comment: "Updated by cert-manager",
	}
	
	body, err := json.Marshal(recordUpdate)
	if err != nil {
		return fmt.Errorf("cannot marshall to json: %v", err)
	}

	// Note: Cette implémentation nécessite de connaître l'ID du record
	// Il faudrait d'abord faire un GET pour récupérer l'ID
	url := fmt.Sprintf("%s/%s/TXT", c.nameShieldRecordsUrl(*domain), *name)
	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	status, responseBody, err := c.doRequest(req, true)
	if err != nil {
		return err
	}

	if status != http.StatusCreated && status != http.StatusOK {
		return fmt.Errorf("failed updating TXT record: %d, %s", status, string(responseBody))
	}

	return nil
}

func (c *NameShieldClient) DeleteTxtRecord(domain *string, name *string) error {
	// curl -X DELETE -H "Content-Type: application/json" \
	//   -H "Authorization: Bearer $APIKEY" \
	//   https://api.nameshield.net/dns/v2/zones/<DOMAIN>/records/<NAME>/<TYPE>
	url := fmt.Sprintf("%s/%s/TXT", c.nameShieldRecordsUrl(*domain), *name)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	status, _, err := c.doRequest(req, false)
	if err != nil {
		return err
	}

	if status != http.StatusOK && status != http.StatusNoContent {
		return fmt.Errorf("failed deleting TXT record: %v", err)
	}

	return nil
}