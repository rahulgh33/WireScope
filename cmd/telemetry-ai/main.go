package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/rahulgh33/wirescope/internal/ai"
)

var (
	serverURL = flag.String("server", "http://localhost:8080", "AI agent server URL")
	apiKey    = flag.String("api-key", "", "API key for authentication")
	oneShot   = flag.String("query", "", "Run a single query and exit")
)

func main() {
	flag.Parse()

	if *apiKey == "" {
		*apiKey = os.Getenv("AI_API_KEY")
		if *apiKey == "" {
			*apiKey = "demo-key" // Default for testing
		}
	}

	if *oneShot != "" {
		if err := query(*oneShot); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	interactive()
}

func interactive() {
	cyan := color.New(color.FgCyan, color.Bold)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)

	cyan.Println("\n╔═══════════════════════════════════════════════════════╗")
	cyan.Println("║     Network Telemetry AI Assistant                   ║")
	cyan.Println("╚═══════════════════════════════════════════════════════╝")
	fmt.Println()
	green.Println("Ask questions about network performance, identify issues,")
	green.Println("and get insights from your telemetry data.")
	fmt.Println()
	yellow.Println("Commands:")
	fmt.Println("  help       - Show this help message")
	fmt.Println("  caps       - Show AI capabilities")
	fmt.Println("  exit/quit  - Exit the CLI")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		cyan.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		switch strings.ToLower(input) {
		case "exit", "quit":
			fmt.Println("Goodbye!")
			return
		case "help":
			printHelp()
			continue
		case "caps", "capabilities":
			showCapabilities()
			continue
		}

		if err := query(input); err != nil {
			color.Red("Error: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
	}
}

func query(userQuery string) error {
	reqBody := ai.QueryRequest{
		Query: userQuery,
		TimeRange: ai.TimeRange{
			Start: time.Now().Add(-24 * time.Hour),
			End:   time.Now(),
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := *serverURL + "/api/v1/ai/query"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", *apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(body))
	}

	var response ai.QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	fmt.Println()
	color.New(color.FgGreen, color.Bold).Println("Response:")
	fmt.Println(response.Response.Text)
	fmt.Println()

	if response.Metadata.QueryTimeMs > 0 {
		color.Yellow("Query time: %dms", response.Metadata.QueryTimeMs)
		fmt.Println()
	}

	return nil
}

func showCapabilities() {
	url := *serverURL + "/api/v1/ai/capabilities"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		color.Red("Error creating request: %v\n", err)
		return
	}

	req.Header.Set("Authorization", *apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		color.Red("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		color.Red("Server error (status %d): %s\n", resp.StatusCode, string(body))
		return
	}

	var caps map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&caps); err != nil {
		color.Red("Error decoding response: %v\n", err)
		return
	}

	fmt.Println()
	color.New(color.FgCyan, color.Bold).Println("AI Agent Capabilities:")
	fmt.Println()

	if capabilities, ok := caps["capabilities"].([]interface{}); ok {
		color.Green("Features:")
		for _, cap := range capabilities {
			fmt.Printf("  • %v\n", cap)
		}
		fmt.Println()
	}

	if metrics, ok := caps["supported_metrics"].([]interface{}); ok {
		color.Green("Supported Metrics:")
		for _, metric := range metrics {
			fmt.Printf("  • %v\n", metric)
		}
		fmt.Println()
	}
}

func printHelp() {
	fmt.Println()
	color.New(color.FgCyan, color.Bold).Println("Example Queries:")
	fmt.Println()
	fmt.Println("  • Which clients had the worst performance today?")
	fmt.Println("  • Show me performance trends for the past week")
	fmt.Println("  • Compare client performance across all targets")
	fmt.Println("  • What is the current error rate?")
	fmt.Println("  • Identify clients with high latency")
	fmt.Println()
	color.Green("Commands:")
	fmt.Println("  help  - Show this help message")
	fmt.Println("  caps  - Show AI capabilities")
	fmt.Println("  exit  - Exit the CLI")
	fmt.Println()
}
