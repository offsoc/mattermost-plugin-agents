// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package semanticcache

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateEmbedder_NoAPIKey(t *testing.T) {
	originalKey := os.Getenv("OPENAI_API_KEY")
	originalURL := os.Getenv("OPENAI_API_URL")
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("OPENAI_API_URL")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		}
		if originalURL != "" {
			os.Setenv("OPENAI_API_URL", originalURL)
		}
	}()

	embedder := createEmbedder()

	embedding, err := embedder("test text")

	assert.Error(t, err)
	assert.Nil(t, embedding)
	assert.Contains(t, err.Error(), "OPENAI_API_KEY not configured")
}

func TestCreateEmbedder_EmptyAPIKey(t *testing.T) {
	originalKey := os.Getenv("OPENAI_API_KEY")
	originalURL := os.Getenv("OPENAI_API_URL")
	os.Setenv("OPENAI_API_KEY", "")
	os.Unsetenv("OPENAI_API_URL")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
		if originalURL != "" {
			os.Setenv("OPENAI_API_URL", originalURL)
		}
	}()

	embedder := createEmbedder()

	embedding, err := embedder("test text")

	assert.Error(t, err)
	assert.Nil(t, embedding)
	assert.Contains(t, err.Error(), "OPENAI_API_KEY not configured")
}

func TestCreateEmbedder_SuccessfulAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Contains(t, r.Header.Get("Authorization"), "Bearer test-key-12345")

		var reqBody map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)
		assert.Equal(t, "test text", reqBody["input"])
		assert.Equal(t, "nomic-embed-text", reqBody["model"])

		embedding := make([]float32, embeddingDimensionsNomic)
		for i := range embedding {
			embedding[i] = 0.001 * float32(i)
		}

		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"embedding": embedding,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "test-key-12345")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	embedder := createEmbedderWithURL(server.URL)

	embedding, err := embedder("test text")

	require.NoError(t, err)
	require.NotNil(t, embedding)
	assert.Len(t, embedding, embeddingDimensionsNomic)
	assert.Equal(t, float32(0.0), embedding[0])
	assert.Equal(t, float32(0.001), embedding[1])
}

func TestCreateEmbedder_APIError401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
	}))
	defer server.Close()

	originalKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "invalid-key")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	embedder := createEmbedderWithURL(server.URL)

	embedding, err := embedder("test")

	assert.Error(t, err)
	assert.Nil(t, embedding)
	assert.Contains(t, err.Error(), "401")
}

func TestCreateEmbedder_APIError429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error": {"message": "Rate limit exceeded"}}`))
	}))
	defer server.Close()

	originalKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	embedder := createEmbedderWithURL(server.URL)

	embedding, err := embedder("test")

	assert.Error(t, err)
	assert.Nil(t, embedding)
	assert.Contains(t, err.Error(), "429")
}

func TestCreateEmbedder_APIError500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": {"message": "Internal server error"}}`))
	}))
	defer server.Close()

	originalKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	embedder := createEmbedderWithURL(server.URL)

	embedding, err := embedder("test")

	assert.Error(t, err)
	assert.Nil(t, embedding)
	assert.Contains(t, err.Error(), "500")
}

func TestCreateEmbedder_NetworkError(t *testing.T) {
	originalKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	embedder := createEmbedderWithURL("http://localhost:1")

	embedding, err := embedder("test")

	assert.Error(t, err)
	assert.Nil(t, embedding)
}

func TestCreateEmbedder_MalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{invalid json`))
	}))
	defer server.Close()

	originalKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	embedder := createEmbedderWithURL(server.URL)

	embedding, err := embedder("test")

	assert.Error(t, err)
	assert.Nil(t, embedding)
}

func TestCreateEmbedder_EmptyDataArray(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]interface{}{
			"data": []map[string]interface{}{},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	embedder := createEmbedderWithURL(server.URL)

	embedding, err := embedder("test")

	assert.Error(t, err)
	assert.Nil(t, embedding)
}

func TestCreateEmbedder_MultipleCalls(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		embedding := make([]float32, embeddingDimensionsNomic)
		for i := range embedding {
			embedding[i] = float32(callCount) * 0.1
		}

		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"embedding": embedding,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	embedder := createEmbedderWithURL(server.URL)

	embedding1, err1 := embedder("first call")
	require.NoError(t, err1)
	require.Len(t, embedding1, embeddingDimensionsNomic)

	embedding2, err2 := embedder("second call")
	require.NoError(t, err2)
	require.Len(t, embedding2, embeddingDimensionsNomic)

	embedding3, err3 := embedder("third call")
	require.NoError(t, err3)
	require.Len(t, embedding3, embeddingDimensionsNomic)

	assert.Equal(t, 3, callCount)
	assert.NotEqual(t, embedding1[0], embedding2[0])
	assert.NotEqual(t, embedding2[0], embedding3[0])
}

func TestCreateEmbedder_LargeInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var reqBody map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&reqBody)
		require.NoError(t, err)

		input := reqBody["input"].(string)
		assert.Greater(t, len(input), 1000)

		embedding := make([]float32, embeddingDimensionsNomic)
		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"embedding": embedding,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	embedder := createEmbedderWithURL(server.URL)

	largeText := ""
	for i := 0; i < 200; i++ {
		largeText += "This is a test sentence for large input handling. "
	}

	embedding, err := embedder(largeText)

	require.NoError(t, err)
	require.NotNil(t, embedding)
	assert.Len(t, embedding, embeddingDimensionsNomic)
}

func TestCreateEmbedder_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(35 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	originalKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	embedder := createEmbedderWithURL(server.URL)

	start := time.Now()
	embedding, err := embedder("test")
	duration := time.Since(start)

	assert.Error(t, err)
	assert.Nil(t, embedding)
	assert.Less(t, duration, 32*time.Second, "Should timeout around 30 seconds")
}

func TestCreateEmbedder_RequestFormat(t *testing.T) {
	var capturedRequest map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedRequest)

		embedding := make([]float32, embeddingDimensionsNomic)
		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"embedding": embedding,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalKey := os.Getenv("OPENAI_API_KEY")
	os.Setenv("OPENAI_API_KEY", "test-key")
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	embedder := createEmbedderWithURL(server.URL)

	_, err := embedder("test input")

	require.NoError(t, err)
	assert.Equal(t, "test input", capturedRequest["input"])
	assert.Equal(t, "nomic-embed-text", capturedRequest["model"])
}

func TestCreateEmbedder_AuthorizationHeader(t *testing.T) {
	var capturedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")

		embedding := make([]float32, embeddingDimensionsNomic)
		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"embedding": embedding,
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalKey := os.Getenv("OPENAI_API_KEY")
	testKey := "test-api-key-abc123"
	os.Setenv("OPENAI_API_KEY", testKey)
	defer func() {
		if originalKey != "" {
			os.Setenv("OPENAI_API_KEY", originalKey)
		} else {
			os.Unsetenv("OPENAI_API_KEY")
		}
	}()

	embedder := createEmbedderWithURL(server.URL)

	_, err := embedder("test")

	require.NoError(t, err)
	assert.Equal(t, "Bearer "+testKey, capturedAuth)
}
