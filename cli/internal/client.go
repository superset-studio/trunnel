package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIClient is a thin wrapper around net/http for calling the Kapstan API.
type APIClient struct {
	serverURL   string
	accessToken string
}

// NewAPIClient creates a client pointing at the given server URL with an optional bearer token.
func NewAPIClient(serverURL, accessToken string) *APIClient {
	return &APIClient{serverURL: serverURL, accessToken: accessToken}
}

func (c *APIClient) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.serverURL+path, reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	return http.DefaultClient.Do(req)
}

// apiError is the standard error body returned by the API.
type apiError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// doJSON performs a request and decodes the response into result.
// On non-2xx status codes it reads the error body and returns a formatted error.
func (c *APIClient) doJSON(method, path string, body, result interface{}) error {
	resp, err := c.doRequest(method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var ae apiError
		if json.Unmarshal(respBody, &ae) == nil && ae.Message != "" {
			return fmt.Errorf("%s (HTTP %d)", ae.Message, resp.StatusCode)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}
	return nil
}

// doNoContent performs a request that expects a 204 No Content response.
func (c *APIClient) doNoContent(method, path string, body interface{}) error {
	resp, err := c.doRequest(method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		var ae apiError
		if json.Unmarshal(respBody, &ae) == nil && ae.Message != "" {
			return fmt.Errorf("%s (HTTP %d)", ae.Message, resp.StatusCode)
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}
