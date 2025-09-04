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
	"time"
)

const (
	openAIEndpoint = "https://api.openai.com/v1/responses"
	modelName      = "gpt-5-mini"
)

type responseReq struct {
	Model          string                 `json:"model"`
	Input          string                 `json:"input"`
	N              int                    `json:"n,omitempty"`
	MaxOutput      int                    `json:"max_output_tokens,omitempty"`
	Temperature    float64                `json:"temperature,omitempty"`
	Text           map[string]any         `json:"text,omitempty"`
}

type responseResp struct {
	ID         string        `json:"id"`
	Object     string        `json:"object"`
	Created    int64         `json:"created"`
	Model      string        `json:"model"`
	Output     []outputItem  `json:"output,omitempty"`
	OutputText string        `json:"output_text,omitempty"`
	Candidates []candidate   `json:"candidates,omitempty"`
}

type outputItem struct {
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

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: ai <task description>\nExample: ai find biggest file here")
		os.Exit(2)
	}

	token := os.Getenv("OPENAI_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "Error: OPENAI_TOKEN not set")
		os.Exit(2)
	}

	task := strings.Join(os.Args[1:], " ")

	contextInfo := gatherContext()
	prompt := buildPrompt(task, contextInfo)

	cmds, err := getCommands(context.Background(), token, prompt, 3)
	if err != nil {
		fmt.Fprintln(os.Stderr, "API error:", err)
		os.Exit(1)
	}
	if len(cmds) == 0 {
		fmt.Fprintln(os.Stderr, "No commands generated")
		os.Exit(1)
	}

	choice, err := selectCommand(cmds)
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

func gatherContext() map[string]string {
	pwd, _ := os.Getwd()
	user := os.Getenv("USER")
	if user == "" {
		user = os.Getenv("USERNAME")
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	tz, _ := time.Now().In(time.Local).Zone()

	return map[string]string{
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
		"shell":     shell,
		"pwd":       pwd,
		"user":      user,
		"timezone":  tz,
		"safe_mode": "on",
		"locale":    os.Getenv("LANG"),
	}
}

func buildPrompt(task string, ctx map[string]string) string {
	var b strings.Builder
	b.WriteString("You are a shell command generator.\n")
	b.WriteString("Output exactly one safe, single-line command for POSIX sh/zsh.\n")
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

func getCommands(ctx context.Context, token, prompt string, n int) ([]string, error) {
	reqBody := responseReq{
		Model:       modelName,
		Input:       prompt,
		N:           n,
		MaxOutput:   128,
		Temperature: 0.2,
		Text: map[string]any{
			"format": "text",
		},
	}

	b, _ := json.Marshal(reqBody)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", openAIEndpoint, bytes.NewReader(b))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+token)

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		data, _ := ioReadAll(resp)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(data))
	}

	var rr responseResp
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		return nil, err
	}

	candidates := extractCandidates(rr)
	if len(candidates) == 0 && rr.OutputText != "" {
		candidates = []string{rr.OutputText}
	}
	if len(candidates) == 0 {
		for _, it := range rr.Output {
			if strings.TrimSpace(it.Text) != "" {
				candidates = append(candidates, it.Text)
			}
		}
	}

	unique := make([]string, 0, len(candidates))
	seen := map[string]struct{}{}
	for _, c := range candidates {
		cmd := sanitizeToSingleCommand(c)
		if cmd == "" {
		 continue
		}
		if _, ok := seen[cmd]; ok {
			continue
		}
		seen[cmd] = struct{}{}
		unique = append(unique, cmd)
	}
	return unique, nil
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
	if len(cmds) == 1 {
		return cmds[0], nil
	}
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
