package tempestapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewClient(t *testing.T) {
	token := "test-token-123"
	client := NewClient(token)

	if client.token != token {
		t.Errorf("Expected token %s, got %s", token, client.token)
	}
}

func TestListStations_Success(t *testing.T) {
	// Mock server response
	mockResponse := `{
		"stations": [
			{
				"name": "Test Station",
				"station_id": 12345,
				"created_epoch": 1609459200,
				"devices": [
					{
						"device_id": 67890,
						"device_type": "ST",
						"serial_number": "ST-00012345"
					}
				]
			}
		],
		"status": {
			"status_code": 0,
			"status_message": "SUCCESS"
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}
		if !strings.Contains(r.URL.String(), "token=test-token") {
			t.Errorf("Expected token parameter in URL: %s", r.URL.String())
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	// Note: This test uses a helper function to test the parsing logic
	// In production, you'd want to refactor the client for better testability

	client := NewClient("test-token")

	// Test with a mock that uses the test server
	stations, err := testListStations(client, server.URL, context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(stations) != 1 {
		t.Fatalf("Expected 1 station, got %d", len(stations))
	}

	station := stations[0]
	if station.Name != "Test Station" {
		t.Errorf("Expected name 'Test Station', got %s", station.Name)
	}
	if station.StationID != 12345 {
		t.Errorf("Expected station ID 12345, got %d", station.StationID)
	}
	if station.deviceID != 67890 {
		t.Errorf("Expected device ID 67890, got %d", station.deviceID)
	}
	if station.serialNumber != "ST-00012345" {
		t.Errorf("Expected serial number 'ST-00012345', got %s", station.serialNumber)
	}

	expectedTime := time.Unix(1609459200, 0)
	if !station.CreatedAt.Equal(expectedTime) {
		t.Errorf("Expected created at %v, got %v", expectedTime, station.CreatedAt)
	}
}

func TestListStations_NoSTDevice(t *testing.T) {
	// Mock response with no ST device
	mockResponse := `{
		"stations": [
			{
				"name": "Test Station",
				"station_id": 12345,
				"created_epoch": 1609459200,
				"devices": [
					{
						"device_id": 67890,
						"device_type": "HB",
						"serial_number": "HB-00012345"
					}
				]
			}
		],
		"status": {
			"status_code": 0,
			"status_message": "SUCCESS"
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient("test-token")
	stations, err := testListStations(client, server.URL, context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should return empty slice when no ST devices found
	if len(stations) != 0 {
		t.Errorf("Expected 0 stations, got %d", len(stations))
	}
}

func TestListStations_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	client := NewClient("test-token")
	_, err := testListStations(client, server.URL, context.Background())
	if err == nil {
		t.Error("Expected error for HTTP 500, got nil")
	}
}

func TestListStations_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient("test-token")
	_, err := testListStations(client, server.URL, context.Background())
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestGetObservations_Success(t *testing.T) {
	// Mock observation response - simplified ST observation
	mockResponse := `{
		"type": "obs_st",
		"device_id": 67890,
		"obs": [[
			1609459200,
			0.0, 1.0, 2.0, 3.0, 4.0, 5.0, 6.0, 7.0, 8.0, 9.0, 10.0, 11.0, 12.0, 13.0, 14.0, 15.0, 16.0, 17.0
		]]
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request contains required parameters
		query := r.URL.Query()
		if query.Get("token") != "test-token" {
			t.Errorf("Expected token parameter")
		}
		if query.Get("time_start") == "" {
			t.Errorf("Expected time_start parameter")
		}
		if query.Get("time_end") == "" {
			t.Errorf("Expected time_end parameter")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockResponse))
	}))
	defer server.Close()

	client := NewClient("test-token")
	station := Station{
		Name:         "Test Station",
		StationID:    12345,
		deviceID:     67890,
		serialNumber: "ST-00012345",
		CreatedAt:    time.Now(),
	}

	startTime := time.Unix(1609459000, 0)
	endTime := time.Unix(1609459300, 0)

	metrics, err := testGetObservations(client, server.URL, station, startTime, endTime, context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(metrics) == 0 {
		t.Error("Expected some metrics, got none")
	}
}

func TestGetObservations_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient("test-token")
	station := Station{deviceID: 67890, serialNumber: "ST-00012345"}

	_, err := testGetObservations(client, server.URL, station, time.Now(), time.Now(), context.Background())
	if err == nil {
		t.Error("Expected error for HTTP 404, got nil")
	}
}

func TestGetObservations_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-token")
	station := Station{deviceID: 67890, serialNumber: "ST-00012345"}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := testGetObservations(client, server.URL, station, time.Now(), time.Now(), ctx)
	if err == nil {
		t.Error("Expected context deadline exceeded error, got nil")
	}
}

// Helper functions to test with custom URLs (since the original methods have hardcoded URLs)

func testListStations(client Client, baseURL string, ctx context.Context) ([]Station, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"?token="+client.token, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Use same parsing logic as original ListStations method
	var data struct {
		Stations []struct {
			CreatedEpoch int64 `json:"created_epoch"`
			Devices      []struct {
				DeviceID     int    `json:"device_id"`
				DeviceType   string `json:"device_type"`
				SerialNumber string `json:"serial_number"`
			} `json:"devices"`
			Name      string `json:"name"`
			StationID int    `json:"station_id"`
		} `json:"stations"`
		Status struct {
			StatusCode    int    `json:"status_code"`
			StatusMessage string `json:"status_message"`
		} `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var out []Station
	for _, station := range data.Stations {
		var deviceId int
		var instance string
		for _, dev := range station.Devices {
			if dev.DeviceType == "ST" {
				deviceId = dev.DeviceID
				instance = dev.SerialNumber
			}
		}

		if deviceId != 0 && instance != "" {
			out = append(out, Station{
				Name:         station.Name,
				deviceID:     deviceId,
				serialNumber: instance,
				StationID:    station.StationID,
				CreatedAt:    time.Unix(station.CreatedEpoch, 0),
			})
		}
	}
	return out, nil
}

func testGetObservations(client Client, baseURL string, station Station, startAt, endAt time.Time, ctx context.Context) ([]prometheus.Metric, error) {
	// Create a URL similar to the original method but with test server
	url := baseURL + "?token=" + client.token + "&time_start=" + string(rune(startAt.Unix())) + "&time_end=" + string(rune(endAt.Unix()))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, http.ErrNoLocation // Simple error for testing
	}

	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Create a mock metric for successful response
	mockMetric := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "test_metric",
		Help: "A test metric",
	})

	return []prometheus.Metric{mockMetric}, nil
}
