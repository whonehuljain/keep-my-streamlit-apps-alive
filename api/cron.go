package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Config struct {
	Apps []string `json:"apps"`
}

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	URL       string `json:"url,omitempty"`
	Status    string `json:"status"`
	Message   string `json:"message"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	// Handle preflight requests
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Verify this is a legitimate cron request (optional security)
	userAgent := r.Header.Get("User-Agent")
	if userAgent != "vercel-cron/1.0" && !strings.Contains(userAgent, "curl") {
		fmt.Printf("Warning: Unexpected User-Agent: %s\n", userAgent)
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("%s | CRON_START | Vercel cron job triggered\n", timestamp)

	// Load configuration
	config, err := loadConfig()
	if err != nil {
		fmt.Printf("%s | CONFIG_ERROR | %v\n", timestamp, err)
		response := map[string]interface{}{
			"success":   false,
			"error":     fmt.Sprintf("Config error: %v", err),
			"timestamp": timestamp,
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Execute wake-up process
	results, err := runWakeScript(config.Apps)

	response := map[string]interface{}{
		"timestamp":  timestamp,
		"apps_count": len(config.Apps),
		"results":    results,
	}

	if err != nil {
		fmt.Printf("%s | CRON_END | FAILED | %v\n", timestamp, err)
		response["success"] = false
		response["error"] = err.Error()
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		fmt.Printf("%s | CRON_END | SUCCESS\n", timestamp)
		response["success"] = true
		response["message"] = "Wake-up process completed"
	}

	json.NewEncoder(w).Encode(response)
}

func loadConfig() (*Config, error) {
	// Load from environment variable (recommended for Vercel)
	appsEnv := os.Getenv("STREAMLIT_APPS")
	if appsEnv != "" {
		var apps []string
		if err := json.Unmarshal([]byte(appsEnv), &apps); err != nil {
			return nil, fmt.Errorf("failed to parse STREAMLIT_APPS env var: %w", err)
		}
		return &Config{Apps: apps}, nil
	}

	// Fallback to hardcoded config (not recommended for production)
	return &Config{
		Apps: []string{
			"https://f1nalyze.streamlit.app/",
			"https://your-other-app.streamlit.app/",
		},
	}, nil
}

func runWakeScript(apps []string) ([]map[string]interface{}, error) {
	results := make([]map[string]interface{}, 0, len(apps))

	// Create the Python script inline for Vercel environment
	script := `#!/usr/bin/env python3
import sys
import subprocess
import time
import json

# Install playwright if not available
try:
    from playwright.sync_api import sync_playwright
except ImportError:
    print("Installing playwright...")
    subprocess.check_call([sys.executable, "-m", "pip", "install", "playwright"])
    subprocess.check_call([sys.executable, "-m", "playwright", "install", "chromium"])
    from playwright.sync_api import sync_playwright

def wake_app(url):
    result = {"url": url, "status": "unknown", "message": ""}
    
    try:
        with sync_playwright() as p:
            browser = p.chromium.launch(
                headless=True,
                args=['--no-sandbox', '--disable-dev-shm-usage']
            )
            page = browser.new_page()
            
            try:
                page.goto(url, timeout=30000, wait_until='networkidle')
                time.sleep(3)
                
                # Look for wake-up buttons
                buttons = [
                    "Yes, get this app back up!",
                    "Wake up",
                    "Start app",
                    "Rerun"
                ]
                
                button_clicked = False
                for btn_text in buttons:
                    try:
                        button = page.locator(f"button:has-text('{btn_text}')")
                        if button.is_visible():
                            button.click()
                            result["status"] = "woken_up"
                            result["message"] = f"Clicked: {btn_text}"
                            button_clicked = True
                            time.sleep(5)
                            break
                    except:
                        continue
                
                if not button_clicked:
                    result["status"] = "already_awake"
                    result["message"] = "No wake-up button found, app appears awake"
                    
            except Exception as e:
                result["status"] = "error"
                result["message"] = str(e)
            finally:
                browser.close()
                
    except Exception as e:
        result["status"] = "error"
        result["message"] = f"Browser error: {str(e)}"
    
    print(json.dumps(result))
    return result

if __name__ == '__main__':
    urls = sys.argv[1:]
    for url in urls:
        wake_app(url)
        time.sleep(2)
`

	// Write script to temporary file
	scriptPath := "/tmp/wake_streamlit.py"
	err := ioutil.WriteFile(scriptPath, []byte(script), 0755)
	if err != nil {
		return results, fmt.Errorf("failed to create script: %w", err)
	}

	// Execute Python script for each app
	for _, app := range apps {
		result := map[string]interface{}{
			"url":     app,
			"status":  "unknown",
			"message": "",
		}

		cmd := exec.Command("python3", scriptPath, app)
		output, err := cmd.CombinedOutput()

		if err != nil {
			result["status"] = "error"
			result["message"] = fmt.Sprintf("Execution error: %v", err)
		} else {
			// Try to parse JSON output from Python script
			outputStr := strings.TrimSpace(string(output))
			lines := strings.Split(outputStr, "\n")

			for _, line := range lines {
				var pythonResult map[string]interface{}
				if json.Unmarshal([]byte(line), &pythonResult) == nil {
					if pythonResult["url"] == app {
						result = pythonResult
						break
					}
				}
			}
		}

		results = append(results, result)
		fmt.Printf("App: %s | Status: %s | Message: %s\n",
			result["url"], result["status"], result["message"])
	}

	return results, nil
}
