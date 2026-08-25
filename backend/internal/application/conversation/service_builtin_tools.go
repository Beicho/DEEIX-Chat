package conversation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/infra/config"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/shared/security"
)

type webSearchArguments struct {
	Query      string `json:"query"`
	MaxResults int    `json:"max_results"`
}

type webSearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type codeSandboxArguments struct {
	Code string `json:"code"`
}

func webSearchAvailable(cfg config.Config) bool {
	switch strings.TrimSpace(cfg.WebSearchProvider) {
	case "searxng", "tavily", "bocha":
		return strings.TrimSpace(cfg.WebSearchBaseURL) != ""
	default:
		return false
	}
}

func (s *Service) executeBuiltInToolCall(ctx context.Context, input ExecuteToolInput) (string, error) {
	switch strings.TrimSpace(input.BuiltInKind) {
	case "web_search":
		return s.executeWebSearchTool(ctx, input.ArgumentsJSON)
	case "run_python":
		return s.executePythonSandboxTool(ctx, input.ArgumentsJSON)
	default:
		return "", fmt.Errorf("tool %s is not enabled for this run", strings.TrimSpace(input.ToolName))
	}
}

func (s *Service) executeWebSearchTool(ctx context.Context, argumentsJSON string) (string, error) {
	cfg := s.cfg.Snapshot()
	if !webSearchAvailable(cfg) {
		return "", fmt.Errorf("web search is not configured")
	}
	var args webSearchArguments
	if err := json.Unmarshal([]byte(strings.TrimSpace(argumentsJSON)), &args); err != nil {
		return "", err
	}
	query := strings.TrimSpace(args.Query)
	if query == "" {
		return "", fmt.Errorf("query is required")
	}
	maxResults := args.MaxResults
	if maxResults <= 0 {
		maxResults = cfg.WebSearchMaxResults
	}
	if maxResults <= 0 {
		maxResults = 5
	}
	if maxResults > 10 {
		maxResults = 10
	}

	timeout := cfg.WebSearchTimeoutSeconds
	if timeout <= 0 {
		timeout = 30
	}
	requestCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	var results []webSearchResult
	var err error
	switch strings.TrimSpace(cfg.WebSearchProvider) {
	case "searxng":
		results, err = s.searchSearXNG(requestCtx, cfg, query, maxResults)
	case "tavily":
		results, err = s.searchTavily(requestCtx, cfg, query, maxResults)
	case "bocha":
		results, err = s.searchBocha(requestCtx, cfg, query, maxResults)
	default:
		err = fmt.Errorf("web search provider is disabled")
	}
	if err != nil {
		return "", err
	}
	payload := map[string]interface{}{
		"query":   query,
		"results": results,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (s *Service) searchSearXNG(ctx context.Context, cfg config.Config, query string, maxResults int) ([]webSearchResult, error) {
	endpoint, err := withQuery(cfg.WebSearchBaseURL, map[string]string{
		"q":      query,
		"format": "json",
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	payload, err := s.doSearchRequest(req, cfg)
	if err != nil {
		return nil, err
	}
	var response struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, err
	}
	results := make([]webSearchResult, 0, minInt(maxResults, len(response.Results)))
	for _, item := range response.Results {
		results = appendSearchResult(results, item.Title, item.URL, item.Content, maxResults)
	}
	return results, nil
}

func (s *Service) searchTavily(ctx context.Context, cfg config.Config, query string, maxResults int) ([]webSearchResult, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"api_key":     strings.TrimSpace(cfg.WebSearchAPIKey),
		"query":       query,
		"max_results": maxResults,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(cfg.WebSearchBaseURL), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	payload, err := s.doSearchRequest(req, cfg)
	if err != nil {
		return nil, err
	}
	var response struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, err
	}
	results := make([]webSearchResult, 0, minInt(maxResults, len(response.Results)))
	for _, item := range response.Results {
		results = appendSearchResult(results, item.Title, item.URL, item.Content, maxResults)
	}
	return results, nil
}

func (s *Service) searchBocha(ctx context.Context, cfg config.Config, query string, maxResults int) ([]webSearchResult, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"query": query,
		"count": maxResults,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimSpace(cfg.WebSearchBaseURL), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if key := strings.TrimSpace(cfg.WebSearchAPIKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	payload, err := s.doSearchRequest(req, cfg)
	if err != nil {
		return nil, err
	}
	var response struct {
		Data struct {
			WebPages struct {
				Value []struct {
					Name    string `json:"name"`
					URL     string `json:"url"`
					Snippet string `json:"snippet"`
					Summary string `json:"summary"`
				} `json:"value"`
			} `json:"webPages"`
		} `json:"data"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, err
	}
	results := make([]webSearchResult, 0, minInt(maxResults, len(response.Data.WebPages.Value)))
	for _, item := range response.Data.WebPages.Value {
		snippet := strings.TrimSpace(item.Snippet)
		if snippet == "" {
			snippet = item.Summary
		}
		results = appendSearchResult(results, item.Name, item.URL, snippet, maxResults)
	}
	return results, nil
}

func (s *Service) doSearchRequest(req *http.Request, cfg config.Config) ([]byte, error) {
	client := security.NewOutboundHTTPClient(
		security.NewStrictOutboundPolicy(cfg.SSRFProtectionEnabled),
		time.Duration(maxInt(cfg.WebSearchTimeoutSeconds, 1))*time.Second,
	)
	if key := strings.TrimSpace(cfg.WebSearchAPIKey); key != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("web search request failed: status=%d", resp.StatusCode)
	}
	return body, nil
}

func (s *Service) executePythonSandboxTool(ctx context.Context, argumentsJSON string) (string, error) {
	cfg := s.cfg.Snapshot()
	if !cfg.CodeSandboxEnabled {
		return "", fmt.Errorf("code execution is not enabled")
	}
	var args codeSandboxArguments
	if err := json.Unmarshal([]byte(strings.TrimSpace(argumentsJSON)), &args); err != nil {
		return "", err
	}
	code := strings.TrimSpace(args.Code)
	if code == "" {
		return "", fmt.Errorf("code is required")
	}
	maxCodeChars := cfg.CodeSandboxMaxCodeChars
	if maxCodeChars <= 0 {
		maxCodeChars = 12000
	}
	if len([]rune(code)) > maxCodeChars {
		return "", fmt.Errorf("code is too long")
	}
	timeout := cfg.CodeSandboxTimeoutSeconds
	if timeout <= 0 {
		timeout = 5
	}
	maxOutputChars := cfg.CodeSandboxMaxOutputChars
	if maxOutputChars <= 0 {
		maxOutputChars = 12000
	}

	workDir, err := os.MkdirTemp("", "deeix-code-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(workDir) //nolint:errcheck

	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "python3", "-I", "-S", "-c", code)
	cmd.Dir = workDir
	cmd.Env = []string{"PYTHONIOENCODING=utf-8"}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()

	exitCode := 0
	if err != nil {
		exitCode = -1
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
	}
	timedOut := runCtx.Err() == context.DeadlineExceeded
	payload := map[string]interface{}{
		"stdout":       truncateRunes(stdout.String(), maxOutputChars),
		"stderr":       truncateRunes(stderr.String(), maxOutputChars),
		"exit_code":    exitCode,
		"timed_out":    timedOut,
		"output_limit": maxOutputChars,
	}
	if err != nil && stderr.Len() == 0 && !timedOut {
		payload["stderr"] = truncateRunes(err.Error(), maxOutputChars)
	}
	raw, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return "", marshalErr
	}
	return string(raw), nil
}

func withQuery(rawURL string, values map[string]string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	for key, value := range values {
		query.Set(key, value)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func appendSearchResult(items []webSearchResult, title string, rawURL string, snippet string, limit int) []webSearchResult {
	if len(items) >= limit {
		return items
	}
	urlValue := strings.TrimSpace(rawURL)
	if urlValue == "" {
		return items
	}
	return append(items, webSearchResult{
		Title:   strings.TrimSpace(title),
		URL:     urlValue,
		Snippet: truncateRunes(strings.TrimSpace(snippet), 500),
	})
}

func truncateRunes(value string, maxChars int) string {
	if maxChars <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxChars {
		return value
	}
	return string(runes[:maxChars])
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}

func maxInt(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
