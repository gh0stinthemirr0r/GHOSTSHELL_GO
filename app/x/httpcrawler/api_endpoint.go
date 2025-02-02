package httpcrawler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

// APIEndpoint manages the HTTP API server
type APIEndpoint struct {
	mutex     sync.Mutex
	results   []Result
	startTime string
}

// Result represents an HTTP probe result
type Result struct {
	URL    string `json:"url"`
	Status int    `json:"status"`
	Body   string `json:"body"`
}

// NewAPIEndpoint creates a new APIEndpoint instance
func NewAPIEndpoint() *APIEndpoint {
	return &APIEndpoint{
		results:   []Result{},
		startTime: "Not started",
	}
}

// Start initializes and starts the HTTP server
func (api *APIEndpoint) Start(port int) {
	http.HandleFunc("/status", api.statusHandler)
	http.HandleFunc("/results", api.resultsHandler)
	http.HandleFunc("/add-result", api.addResultHandler)

	serverAddress := fmt.Sprintf(":%d", port)
	fmt.Printf("API server is running on %s\n", serverAddress)
	if err := http.ListenAndServe(serverAddress, nil); err != nil {
		fmt.Printf("Error starting API server: %v\n", err)
	}
}

// statusHandler handles the /status endpoint
func (api *APIEndpoint) statusHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"status":     "running",
		"start_time": api.startTime,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// resultsHandler handles the /results endpoint
func (api *APIEndpoint) resultsHandler(w http.ResponseWriter, r *http.Request) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(api.results)
}

// addResultHandler handles adding results via POST to /add-result
func (api *APIEndpoint) addResultHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var result Result
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	api.mutex.Lock()
	api.results = append(api.results, result)
	api.mutex.Unlock()

	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write([]byte("Result added successfully"))
}
