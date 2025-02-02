package output

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/jung-kurt/gofpdf"
)

// ReportData represents the data to be included in the reports.
type ReportData struct {
	Source           string
	Destination      string
	OS               string
	OpenPorts        []int
	Shares           []string
	GeneratedTraffic int
	Metrics          map[string]string
}

// WriteCSVReport generates a detailed CSV report.
func WriteCSVReport(data []ReportData, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Source", "Destination", "OS", "Open Ports", "Shares", "Generated Traffic", "Metrics"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, entry := range data {
		row := []string{
			entry.Source,
			entry.Destination,
			entry.OS,
			fmt.Sprintf("%v", entry.OpenPorts),
			fmt.Sprintf("%v", entry.Shares),
			fmt.Sprintf("%d", entry.GeneratedTraffic),
			fmt.Sprintf("%v", entry.Metrics),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	return nil
}

// WritePDFReport generates a detailed PDF report.
func WritePDFReport(data []ReportData, filePath string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Arial", "B", 14)
	pdf.AddPage()

	pdf.Cell(40, 10, "Traffic Analysis Report")
	pdf.Ln(12)
	pdf.SetFont("Arial", "", 12)

	for _, entry := range data {
		pdf.Cell(40, 10, fmt.Sprintf("Source: %s", entry.Source))
		pdf.Ln(6)
		pdf.Cell(40, 10, fmt.Sprintf("Destination: %s", entry.Destination))
		pdf.Ln(6)
		pdf.Cell(40, 10, fmt.Sprintf("OS: %s", entry.OS))
		pdf.Ln(6)
		pdf.Cell(40, 10, fmt.Sprintf("Open Ports: %v", entry.OpenPorts))
		pdf.Ln(6)
		pdf.Cell(40, 10, fmt.Sprintf("Shares: %v", entry.Shares))
		pdf.Ln(6)
		pdf.Cell(40, 10, fmt.Sprintf("Generated Traffic: %d packets", entry.GeneratedTraffic))
		pdf.Ln(6)
		pdf.Cell(40, 10, "Metrics:")
		pdf.Ln(6)
		for key, value := range entry.Metrics {
			pdf.Cell(40, 10, fmt.Sprintf("- %s: %s", key, value))
			pdf.Ln(6)
		}
		pdf.Ln(12)
	}

	if err := pdf.OutputFileAndClose(filePath); err != nil {
		return fmt.Errorf("failed to write PDF report: %w", err)
	}

	return nil
}

// PrintConsoleReport outputs a summary to the console.
func PrintConsoleReport(data []ReportData) {
	fmt.Println("\nTraffic Analysis Report:")
	fmt.Println("-----------------------")
	for _, entry := range data {
		fmt.Printf("Source: %s\n", entry.Source)
		fmt.Printf("Destination: %s\n", entry.Destination)
		fmt.Printf("OS: %s\n", entry.OS)
		fmt.Printf("Open Ports: %v\n", entry.OpenPorts)
		fmt.Printf("Shares: %v\n", entry.Shares)
		fmt.Printf("Generated Traffic: %d packets\n", entry.GeneratedTraffic)
		fmt.Println("Metrics:")
		for key, value := range entry.Metrics {
			fmt.Printf("  - %s: %s\n", key, value)
		}
		fmt.Println("-----------------------")
	}
}
