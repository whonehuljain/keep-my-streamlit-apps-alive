package executor

import (
	"context"
	"fmt"
	"keep-streamlit-alive/internal/config"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type PythonExecutor struct {
	scriptPath string
	pythonPath string
	timeout    time.Duration
}

func NewPythonExecutor(scriptPath string, timeout time.Duration) *PythonExecutor {
	// find Python executable
	pythonPath := findPythonExecutable()

	return &PythonExecutor{
		scriptPath: scriptPath,
		pythonPath: pythonPath,
		timeout:    timeout,
	}
}

func findPythonExecutable() string {
	candidates := []string{"python3", "python", "py"}

	for _, candidate := range candidates {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}

	return "python3"
}

func (pe *PythonExecutor) ExecuteWakeUpScript(apps []config.StreamlitApp) error {
	fmt.Printf("Starting wake-up process for %d apps...\n", len(apps))

	// create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), pe.timeout)
	defer cancel()

	// prep app URLs as command line args
	var urls []string
	for _, app := range apps {
		urls = append(urls, app.URL)
		fmt.Printf("  - %s: %s\n", app.Name, app.URL)
	}

	// build command args
	args := []string{pe.scriptPath}
	args = append(args, urls...)

	// execute
	cmd := exec.CommandContext(ctx, pe.pythonPath, args...)

	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()

	// Print the output from Python script
	if len(output) > 0 {
		fmt.Printf("Python script output:\n%s\n", string(output))
	}

	if err != nil {
		// Check if it was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("python script execution timed out after %v", pe.timeout)
		}
		return fmt.Errorf("python script execution failed: %w\nOutput: %s", err, string(output))
	}

	fmt.Println("Wake-up process completed successfully!")
	return nil
}

func (pe *PythonExecutor) ValidateScript() error {
	// Check if script exists
	if _, err := os.Stat(pe.scriptPath); os.IsNotExist(err) {
		return fmt.Errorf("python script not found: %s", pe.scriptPath)
	}

	// Check if Python is available
	cmd := exec.Command(pe.pythonPath, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("python executable not found or not working: %s", pe.pythonPath)
	}

	return nil
}

// InstallPythonDependencies installs required Python packages
func (pe *PythonExecutor) InstallPythonDependencies() error {
	fmt.Println("Installing Python dependencies...")

	dependencies := []string{"playwright", "requests"}

	for _, dep := range dependencies {
		fmt.Printf("Installing %s...\n", dep)
		cmd := exec.Command(pe.pythonPath, "-m", "pip", "install", dep)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to install %s: %w\nOutput: %s", dep, err, string(output))
		}
	}

	// Install Playwright browsers
	fmt.Println("Installing Playwright browsers...")
	cmd := exec.Command(pe.pythonPath, "-m", "playwright", "install", "chromium")
	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Printf("Warning: Failed to install Playwright browsers: %v\nOutput: %s\n", err, string(output))
		// Don't return error as this might work in some environments
	}

	fmt.Println("Python dependencies installed successfully!")
	return nil
}

// GetScriptPath returns the absolute path to the Python script
func (pe *PythonExecutor) GetScriptPath() (string, error) {
	return filepath.Abs(pe.scriptPath)
}
