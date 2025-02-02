package ghostsuite

import "fmt"

// App represents a GHOSTSHELL app with metadata.
type App struct {
	Name        string // Name of the application
	Description string // Brief description of the application
}

// GetApps returns a list of all GHOSTSHELL applications.
func GetApps() []App {
	return []App{
		{"ASN Scanner", "Scans IPs, domains, or ASNs for network information."},
		{"TLD Finder", "Finds top-level domains and provides insights."},
		{"URL Crawler", "Crawls and extracts URLs for further analysis."},
		{"Web Crawler", "Performs deep web crawling for target enumeration."},
		// Add more apps as needed
	}
}

// PrintApps outputs the app list to the console.
func PrintApps() {
	fmt.Println("GHOSTSHELL App Suite:")
	for _, app := range GetApps() {
		fmt.Printf("- %s: %s\n", app.Name, app.Description)
	}
}
