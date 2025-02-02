package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ModelInfo represents the structure of a single model entry returned by the Hugging Face search API.
// This structure is simplified for illustration; real API JSON has more fields you may want to parse.
type ModelInfo struct {
	ModelID  string `json:"modelId"`
	Siblings []struct {
		Rfilename string `json:"rfilename"`
	} `json:"siblings"`
	Tags      []string `json:"tags"`
	Likes     int      `json:"likes"`
	Downloads int      `json:"downloads"`
}

// Global variables
var (
	searchQuery          string
	availableModels      []ModelInfo
	selectedIndex        int
	logger               *zap.Logger
	downloadProgress     int64 // used atomically to store the current bytes downloaded
	downloadTotal        int64 // total bytes expected
	isDownloading        bool
	downloadError        error // set if there's an error during download
	currentDownloadingID string
)

// setupLogger configures our dynamic logging. We'll create a filename that includes the UTC date/time.
// We'll also use a custom Zap production config that ensures time is logged in UTC ISO8601 format.
func setupLogger() (*zap.Logger, error) {
	currentTime := time.Now().UTC()
	logFileName := fmt.Sprintf("ai_%s.log", currentTime.Format("2006-01-02_15-04-05"))

	// Create a custom production config
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder // Use ISO8601 for UTC timestamps
	// Log to both stdout and the dynamically created file
	cfg.OutputPaths = []string{"stdout", logFileName}
	// Build logger
	return cfg.Build()
}

// searchHuggingFace queries the Hugging Face API for models matching the provided query string.
// This function returns a slice of ModelInfo structs, filtered for those that have .gguf files.
func searchHuggingFace(query string, logger *zap.Logger) ([]ModelInfo, error) {
	// Basic public endpoint for searching models:
	// GET https://huggingface.co/api/models?search=<query>
	url := fmt.Sprintf("https://huggingface.co/api/models?search=%s", query)

	logger.Info("Searching Hugging Face", zap.String("url", url))
	resp, err := http.Get(url)
	if err != nil {
		logger.Error("Request to Hugging Face failed", zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("Hugging Face returned non-OK status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("status", resp.Status),
		)
		return nil, fmt.Errorf("non-OK status: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Failed to read response body", zap.Error(err))
		return nil, err
	}

	var allModels []ModelInfo
	if err := json.Unmarshal(body, &allModels); err != nil {
		logger.Error("Failed to parse JSON", zap.Error(err))
		return nil, err
	}

	// Filter down to only those that appear to have one or more .gguf files
	var ggufModels []ModelInfo
	for _, m := range allModels {
		for _, sibling := range m.Siblings {
			if strings.HasSuffix(sibling.Rfilename, ".gguf") {
				ggufModels = append(ggufModels, m)
				break // once found, no need to scan other siblings
			}
		}
	}

	logger.Info("Search complete", zap.Int("total_models_found", len(ggufModels)))
	return ggufModels, nil
}

// fetchContentLength attempts a HEAD request to get the file size from the server, if provided.
func fetchContentLength(fileURL string, logger *zap.Logger) int64 {
	req, err := http.NewRequest(http.MethodHead, fileURL, nil)
	if err != nil {
		logger.Warn("HEAD request creation failed", zap.Error(err))
		return -1
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Warn("HEAD request failed", zap.String("url", fileURL), zap.Error(err))
		return -1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Warn("HEAD request non-OK status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("status", resp.Status),
		)
		return -1
	}

	contentLength := resp.ContentLength
	logger.Debug("HEAD request success", zap.Int64("content_length", contentLength))
	return contentLength
}

// downloadModel fetches a single file (e.g., .gguf) from a direct URL, showing download progress, and saves it to 'savePath'.
func downloadModel(fileURL, savePath string, logger *zap.Logger) error {
	// Attempt HEAD to get content length
	totalBytes := fetchContentLength(fileURL, logger)
	if totalBytes <= 0 {
		logger.Warn("Server did not provide content length, progress bar will be approximate.")
	}

	// Perform the actual GET
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, fileURL, nil)
	if err != nil {
		logger.Error("Failed to create GET request", zap.Error(err))
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error("Failed to download .gguf file", zap.String("url", fileURL), zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Error("Failed to download .gguf file, non-OK status",
			zap.String("status", resp.Status),
			zap.String("url", fileURL),
		)
		return fmt.Errorf("failed to download file: %s", resp.Status)
	}

	// Setup progress counters
	atomic.StoreInt64(&downloadProgress, 0)
	atomic.StoreInt64(&downloadTotal, totalBytes)

	// Create local file
	out, err := os.Create(savePath)
	if err != nil {
		logger.Error("Failed to create output file", zap.Error(err))
		return err
	}
	defer out.Close()

	// Copy with a progress tracking wrapper
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			// Write to file
			if _, wErr := out.Write(buf[:n]); wErr != nil {
				logger.Error("Failed to write to output file", zap.Error(wErr))
				return wErr
			}
			// Update progress
			atomic.AddInt64(&downloadProgress, int64(n))
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				// We reached the end
				break
			}
			logger.Error("Read error during download", zap.Error(readErr))
			return readErr
		}
	}

	logger.Info("Downloaded .gguf successfully", zap.String("path", savePath))
	return nil
}

// downloadFirstGGUF tries to find a .gguf file in the model's siblings array. If found,
// it constructs the direct download link, then calls downloadModel to store it locally in ./ai/models.
func downloadFirstGGUF(model ModelInfo, modelsPath string, logger *zap.Logger) error {
	if len(model.Siblings) == 0 {
		return fmt.Errorf("no siblings in model: %s", model.ModelID)
	}

	var ggufFileName string
	for _, s := range model.Siblings {
		if strings.HasSuffix(s.Rfilename, ".gguf") {
			ggufFileName = s.Rfilename
			break
		}
	}
	if ggufFileName == "" {
		return fmt.Errorf("no .gguf file found for model: %s", model.ModelID)
	}

	// Construct direct link for .gguf
	fileURL := fmt.Sprintf("https://huggingface.co/%s/resolve/main/%s", model.ModelID, ggufFileName)
	savePath := filepath.Join(modelsPath, ggufFileName)

	logger.Info("Attempting to download .gguf file",
		zap.String("model_id", model.ModelID),
		zap.String("file_url", fileURL),
	)

	isDownloading = true
	currentDownloadingID = model.ModelID
	downloadError = nil
	err := downloadModel(fileURL, savePath, logger)
	if err != nil {
		downloadError = err
	}
	isDownloading = false
	currentDownloadingID = ""

	return err
}

func main() {
	var err error
	logger, err = setupLogger()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	modelsPath := "./ai/models"
	if err := os.MkdirAll(modelsPath, os.ModePerm); err != nil {
		logger.Fatal("Failed to create models directory", zap.Error(err))
	}

	availableModels, _ = searchHuggingFace("gguf", logger)
	searchQuery = "gguf"

	fmt.Println("Model Navigator initialized.")
}
