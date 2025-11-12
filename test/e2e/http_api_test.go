// +build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	baseURL = "http://localhost:8080"
)

// TestHTTPAPIEndToEnd는 HTTP API의 전체 플로우를 테스트합니다
func TestHTTPAPIEndToEnd(t *testing.T) {
	// 서비스가 실행 중인지 확인
	resp, err := http.Get(baseURL + "/health")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.Status Code)

	t.Run("Complete CRUD Flow", func(t *testing.T) {
		var documentID string

		// 1. Create Document
		t.Log("Creating document...")
		createPayload := map[string]interface{}{
			"collection": "e2e_test",
			"data": map[string]interface{}{
				"title":       "E2E Test Document",
				"description": "Created by end-to-end test",
				"timestamp":   time.Now().Unix(),
			},
		}

		createResp, err := makeRequest("POST", "/api/v1/documents", createPayload)
		require.NoError(t, err)
		assert.Equal(t, http.StatusCreated, createResp.StatusCode)

		var createResult map[string]interface{}
		err = json.NewDecoder(createResp.Body).Decode(&createResult)
		require.NoError(t, err)
		documentID = createResult["id"].(string)
		assert.NotEmpty(t, documentID)

		// 2. Get Document
		t.Log("Fetching document...")
		getResp, err := makeRequest("GET", fmt.Sprintf("/api/v1/documents/e2e_test/%s", documentID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, getResp.StatusCode)

		var getResult map[string]interface{}
		err = json.NewDecoder(getResp.Body).Decode(&getResult)
		require.NoError(t, err)
		assert.Equal(t, "E2E Test Document", getResult["data"].(map[string]interface{})["title"])

		// 3. Update Document
		t.Log("Updating document...")
		updatePayload := map[string]interface{}{
			"data": map[string]interface{}{
				"title":       "Updated E2E Test Document",
				"description": "Updated by end-to-end test",
				"updated_at":  time.Now().Unix(),
			},
		}

		updateResp, err := makeRequest("PUT", fmt.Sprintf("/api/v1/documents/e2e_test/%s", documentID), updatePayload)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, updateResp.StatusCode)

		// 4. Verify Update
		t.Log("Verifying update...")
		getUpdatedResp, err := makeRequest("GET", fmt.Sprintf("/api/v1/documents/e2e_test/%s", documentID), nil)
		require.NoError(t, err)

		var getUpdatedResult map[string]interface{}
		err = json.NewDecoder(getUpdatedResp.Body).Decode(&getUpdatedResult)
		require.NoError(t, err)
		assert.Equal(t, "Updated E2E Test Document", getUpdatedResult["data"].(map[string]interface{})["title"])

		// 5. List Documents
		t.Log("Listing documents...")
		listResp, err := makeRequest("GET", "/api/v1/documents/e2e_test?limit=10", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, listResp.StatusCode)

		var listResult map[string]interface{}
		err = json.NewDecoder(listResp.Body).Decode(&listResult)
		require.NoError(t, err)
		assert.Greater(t, len(listResult["documents"].([]interface{})), 0)

		// 6. Delete Document
		t.Log("Deleting document...")
		deleteResp, err := makeRequest("DELETE", fmt.Sprintf("/api/v1/documents/e2e_test/%s", documentID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, deleteResp.StatusCode)

		// 7. Verify Deletion
		t.Log("Verifying deletion...")
		getDeletedResp, err := makeRequest("GET", fmt.Sprintf("/api/v1/documents/e2e_test/%s", documentID), nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, getDeletedResp.StatusCode)
	})

	t.Run("Health Check Endpoints", func(t *testing.T) {
		// /health
		healthResp, err := makeRequest("GET", "/health", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, healthResp.StatusCode)

		// /ready
		readyResp, err := makeRequest("GET", "/ready", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, readyResp.StatusCode)

		// /metrics
		metricsResp, err := makeRequest("GET", "/metrics", nil)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, metricsResp.StatusCode)
	})
}

// Helper function to make HTTP requests
func makeRequest(method, path string, payload interface{}) (*http.Response, error) {
	var body bytes.Buffer
	if payload != nil {
		json.NewEncoder(&body).Encode(payload)
	}

	req, err := http.NewRequest(method, baseURL+path, &body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tenant-ID", "default")

	client := &http.Client{Timeout: 10 * time.Second}
	return client.Do(req)
}
