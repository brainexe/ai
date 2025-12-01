package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	openAIEndpoint = "https://api.openai.com/v1/responses"
	modelName      = "gpt-5.1"
)

type responseReq struct {
	Model     string         `json:"model"`
	Input     string         `json:"input"`
	MaxOutput int            `json:"max_output_tokens,omitempty"`
	Text      map[string]any `json:"text,omitempty"`
	Reasoning map[string]any `json:"reasoning,omitempty"`
}

type responseResp struct {
	ID         string       `json:"id"`
	Object     string       `json:"object"`
	Created    int64        `json:"created"`
	Model      string       `json:"model"`
	Output     []outputItem `json:"output,omitempty"`
	OutputText string       `json:"output_text,omitempty"`
	Candidates []candidate  `json:"candidates,omitempty"`
}

type outputItem struct {
	Type    string        `json:"type,omitempty"`
	Text    string        `json:"text,omitempty"`
	Content []contentPart `json:"content,omitempty"`
	Role    string        `json:"role,omitempty"`
}

type contentPart struct {
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
}

type candidate struct {
	Content candidateContent `json:"content"`
}

type candidateContent struct {
	Type  string          `json:"type,omitempty"`
	Parts []candidatePart `json:"parts,omitempty"`
}

type candidatePart struct {
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
}

type apiCallResult struct {
	Commands    []string        `json:"commands"`
	Duration    time.Duration   `json:"duration"`
	RawResponse json.RawMessage `json:"raw_response"`
	Error       error           `json:"error,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ai [-v] [-n <number>] <task description>\nExample: ai find biggest file here\n       ai -v list files in current dir\n       ai -n 5 find files here")
		os.Exit(2)
	}

	// Parse flags
	var verbose bool
	var numCommands = 3 // default
	var taskStart = 1

	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch arg {
		case "-v":
			verbose = true
			taskStart = i + 1
		case "-n":
			if i+1 >= len(os.Args) {
				fmt.Fprintln(os.Stderr, "Error: -n requires a number argument")
				os.Exit(2)
			}
			var err error
			numCommands, err = strconv.Atoi(os.Args[i+1])
			if err != nil || numCommands < 1 {
				fmt.Fprintln(os.Stderr, "Error: -n requires a positive integer")
				os.Exit(2)
			}
			i++ // skip the number argument
			taskStart = i + 1
		default:
			// First non-flag argument starts the task description
			taskStart = i
			break
		}
	}

	if taskStart >= len(os.Args) {
		fmt.Fprintln(os.Stderr, "Usage: ai [-v] [-n <number>] <task description>")
		os.Exit(2)
	}

	token := os.Getenv("OPENAI_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "Error: OPENAI_TOKEN not set")
		os.Exit(2)
	}

	task := strings.Join(os.Args[taskStart:], " ")

	contextInfo := gatherContext()
	prompt := buildPrompt(task, contextInfo)

	results, err := getCommands(context.Background(), token, prompt, verbose, numCommands)
	if err != nil {
		fmt.Fprintln(os.Stderr, "API error:", err)
		os.Exit(1)
	}
	if len(results) == 0 || len(results[0].Commands) == 0 {
		fmt.Fprintln(os.Stderr, "No commands generated")
		os.Exit(1)
	}

	// Show verbose output if requested
	if verbose {
		printVerboseOutput(results)
	}

	// Use the first result (combined/aggregated) for command selection
	choice, err := selectCommand(results[0].Commands)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Selection error:", err)
		os.Exit(1)
	}

	// Echo the command for transparency
	fmt.Println(choice)

	// Execute with inherited stdio so it behaves like calling directly
	if err := runCommand(choice); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, "Execution error:", err)
		os.Exit(1)
	}
}

func printVerboseOutput(results []apiCallResult) {
	if len(results) == 0 {
		return
	}

	combinedResult := results[0]     // First result is the combined/aggregated result
	individualResults := results[1:] // Rest are individual API call results

	fmt.Println("=== VERBOSE OUTPUT ===")
	fmt.Printf("Commands generated: %d\n", len(combinedResult.Commands))

	// Show timing information
	if len(individualResults) > 0 {
		fmt.Printf("Total API request time: %v\n", combinedResult.Duration)
		fmt.Printf("Number of concurrent API calls: %d\n", len(individualResults))
		fmt.Printf("Average API request time: %v\n", combinedResult.Duration/time.Duration(len(individualResults)))
	} else {
		fmt.Printf("API request time: %v\n", combinedResult.Duration)
	}

	// Show the generated commands
	fmt.Println("\nGenerated commands:")
	for i, cmd := range combinedResult.Commands {
		fmt.Printf("  %d) %s\n", i+1, cmd)
	}

	// Show raw API responses from individual calls if available
	rawResponses := 0
	for i, r := range individualResults {
		if len(r.RawResponse) > 0 {
			rawResponses++
			fmt.Printf("\nAPI Call %d Response (pretty-printed):\n", i+1)
			var prettyJSON bytes.Buffer
			if err := json.Indent(&prettyJSON, r.RawResponse, "", "  "); err == nil {
				fmt.Println(prettyJSON.String())
			} else {
				fmt.Println(string(r.RawResponse))
			}
		}
	}

	if rawResponses == 0 {
		fmt.Println("\nNote: Raw API responses not captured (may be due to error or non-verbose mode)")
	}

	fmt.Println("=== END VERBOSE OUTPUT ===")
}

func gatherContext() map[string]string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	systemInfo := readSystemInfo()

	return map[string]string{
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
		"shell":     shell,
		"safe_mode": "on",
		"system":    systemInfo,
	}
}

func readSystemInfo() string {
	data, err := os.ReadFile("/etc/issue")
	if err != nil {
		return ""
	}
	content := string(data)

	// Strip \n \l and extra whitespace
	content = strings.ReplaceAll(content, "\\n", "")
	content = strings.ReplaceAll(content, "\\l", "")
	content = strings.TrimSpace(content)

	return content
}

func buildPrompt(task string, ctx map[string]string) string {
	var b strings.Builder
	b.WriteString("You are a shell command generator.\n")
	b.WriteString("Output exactly one safe, single-line command for POSIX " + ctx["shell"] + "\n")
	b.WriteString("Rules:\n")
	b.WriteString("- NO explanations or extra text. Only the command.\n")
	b.WriteString("- Avoid destructive actions (rm -rf, chmod -R, sudo, moving/deleting) unless explicitly requested.\n")
	b.WriteString("- Prefer read-only queries (ls/find/stat/du/grep) when unsure.\n")
	b.WriteString("- Use utilities commonly available on Linux/macOS.\n")
	b.WriteString("- Must run correctly in the current working directory.\n")
	b.WriteString("- If paths contain spaces, quote them safely.\n")
	b.WriteString("- If the task is ambiguous, choose the safest widely useful command.\n")
	b.WriteString("\nEnvironment context:\n")
	for k, v := range ctx {
		if v == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("- %s: %s\n", k, v))
	}
	b.WriteString("\nTask:\n")
	b.WriteString(task)
	b.WriteString("\n")
	return b.String()
}

func getCommands(ctx context.Context, token, prompt string, verbose bool, numCommands int) ([]apiCallResult, error) {
	numConcurrentCalls := numCommands

	type apiResult struct {
		result apiCallResult
		err    error
	}

	results := make(chan apiResult, numConcurrentCalls)
	var wg sync.WaitGroup

	// Function to make a single API call
	makeAPICall := func() {
		defer wg.Done()
		startTime := time.Now()

		reqBody := responseReq{
			Model:     modelName,
			Input:     prompt,
			MaxOutput: 500,
			Text: map[string]any{
				"format": map[string]any{
					"type": "text",
				},
			},
			Reasoning: map[string]any{
				"effort": "none",
			},
		}

		b, _ := json.Marshal(reqBody)
		httpReq, _ := http.NewRequestWithContext(ctx, "POST", openAIEndpoint, bytes.NewReader(b))
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+token)

		httpClient := &http.Client{Timeout: 30 * time.Second}
		resp, err := httpClient.Do(httpReq)
		if err != nil {
			results <- apiResult{apiCallResult{Error: err, Duration: time.Since(startTime)}, err}
			return
		}
		defer resp.Body.Close()

		// Read the full response body for verbose mode or error handling
		var rawResponse json.RawMessage
		var respData []byte
		if verbose {
			if data, err := ioReadAll(resp); err == nil {
				respData = data
				rawResponse = data
			}
		} else {
			// For non-verbose mode, still read the body for error handling
			if data, err := ioReadAll(resp); err == nil {
				respData = data
			}
		}

		if resp.StatusCode >= 400 {
			err := fmt.Errorf("status %d: %s", resp.StatusCode, string(respData))
			results <- apiResult{apiCallResult{Error: err, Duration: time.Since(startTime), RawResponse: rawResponse}, err}
			return
		}

		var rr responseResp
		if respData != nil {
			// Use the already read data
			if err := json.Unmarshal(respData, &rr); err != nil {
				results <- apiResult{apiCallResult{Error: err, Duration: time.Since(startTime), RawResponse: rawResponse}, err}
				return
			}
		} else {
			// Fallback to streaming decode if we didn't read the data
			if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
				results <- apiResult{apiCallResult{Error: err, Duration: time.Since(startTime), RawResponse: rawResponse}, err}
				return
			}
		}

		candidates := extractCandidates(rr)
		if len(candidates) == 0 && rr.OutputText != "" {
			candidates = []string{rr.OutputText}
		}
		if len(candidates) == 0 {
			for _, it := range rr.Output {
				if strings.TrimSpace(it.Text) != "" {
					candidates = append(candidates, it.Text)
				} else if len(it.Content) > 0 {
					for _, part := range it.Content {
						if strings.TrimSpace(part.Text) != "" {
							candidates = append(candidates, part.Text)
						}
					}
				}
			}
		}

		var commands []string
		for _, c := range candidates {
			cmd := sanitizeToSingleCommand(c)
			if cmd != "" {
				commands = append(commands, cmd)
			}
		}

		results <- apiResult{apiCallResult{Commands: commands, Duration: time.Since(startTime), RawResponse: rawResponse}, nil}
	}

	// Launch concurrent API calls
	for i := 0; i < numConcurrentCalls; i++ {
		wg.Add(1)
		go makeAPICall()
	}

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect all results
	var allResults []apiCallResult
	var firstError error

	for result := range results {
		if result.err != nil && firstError == nil {
			firstError = result.err
		}
		allResults = append(allResults, result.result)
	}

	if firstError != nil {
		return nil, firstError
	}

	// Deduplicate commands across all successful results
	unique := make([]string, 0)
	seen := map[string]struct{}{}
	for _, result := range allResults {
		for _, cmd := range result.Commands {
			if _, ok := seen[cmd]; !ok {
				seen[cmd] = struct{}{}
				unique = append(unique, cmd)
			}
		}
	}

	// Create a combined result with all unique commands and total duration
	totalDuration := time.Duration(0)
	for _, result := range allResults {
		totalDuration += result.Duration
	}

	combinedResult := apiCallResult{
		Commands:    unique,
		Duration:    totalDuration,
		RawResponse: nil, // Will show individual responses in verbose output
	}

	// Return the combined result plus all individual results
	return append([]apiCallResult{combinedResult}, allResults...), nil
}

func extractCandidates(rr responseResp) []string {
	var out []string
	for _, c := range rr.Candidates {
		for _, p := range c.Content.Parts {
			if strings.TrimSpace(p.Text) != "" {
				out = append(out, p.Text)
			}
		}
	}
	if len(out) == 0 && rr.OutputText != "" {
		out = append(out, rr.OutputText)
	}
	return out
}

var codeBlockRe = regexp.MustCompile("(?s)```(?:sh|bash|zsh)?\\n(.*?)\\n```")
var firstLineRe = regexp.MustCompile(`(?m)^[^\n#;][^\n]*`)

func sanitizeToSingleCommand(s string) string {
	trim := strings.TrimSpace(s)

	if m := codeBlockRe.FindStringSubmatch(trim); len(m) == 2 {
		trim = strings.TrimSpace(m[1])
	}

	if m := firstLineRe.FindString(trim); m != "" {
		trim = strings.TrimSpace(m)
	}

	if i := strings.IndexByte(trim, '\n'); i >= 0 {
		trim = strings.TrimSpace(trim[:i])
	}

	trim = strings.TrimPrefix(trim, "$ ")
	trim = strings.TrimPrefix(trim, "> ")
	trim = strings.TrimSpace(trim)

	return trim
}

func selectCommand(cmds []string) (string, error) {
	fmt.Println("Select a command:")
	for i, c := range cmds {
		fmt.Printf("  %d) %s\n", i+1, c)
	}
	fmt.Print("Enter number: ")
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(line)
	idx, err := strconv.Atoi(line)
	if err != nil || idx < 1 || idx > len(cmds) {
		return "", errors.New("invalid selection")
	}
	return cmds[idx-1], nil
}

func runCommand(command string) error {
	sh := os.Getenv("SHELL")
	if sh == "" {
		sh = "sh"
	}
	cmd := exec.Command(sh, "-c", command)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	return cmd.Run()
}
func ioReadAll(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, err := buf.ReadFrom(resp.Body)
	return buf.Bytes(), err
}
