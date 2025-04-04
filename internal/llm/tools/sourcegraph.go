package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	SourcegraphToolName        = "sourcegraph"
	sourcegraphToolDescription = `Search code across public repositories using Sourcegraph's GraphQL API.

WHEN TO USE THIS TOOL:
- Use when you need to find code examples or implementations across public repositories
- Helpful for researching how others have solved similar problems
- Useful for discovering patterns and best practices in open source code

HOW TO USE:
- Provide a search query using Sourcegraph's query syntax
- Optionally specify the number of results to return (default: 10)
- Optionally set a timeout for the request

QUERY SYNTAX:
- Basic search: "fmt.Println" searches for exact matches
- File filters: "file:.go fmt.Println" limits to Go files
- Repository filters: "repo:^github\.com/golang/go$ fmt.Println" limits to specific repos
- Language filters: "lang:go fmt.Println" limits to Go code
- Boolean operators: "fmt.Println AND log.Fatal" for combined terms
- Regular expressions: "fmt\.(Print|Printf|Println)" for pattern matching
- Quoted strings: "\"exact phrase\"" for exact phrase matching
- Exclude filters: "-file:test" or "-repo:forks" to exclude matches

ADVANCED FILTERS:
- Repository filters:
  * "repo:name" - Match repositories with name containing "name"
  * "repo:^github\.com/org/repo$" - Exact repository match
  * "repo:org/repo@branch" - Search specific branch
  * "repo:org/repo rev:branch" - Alternative branch syntax
  * "-repo:name" - Exclude repositories
  * "fork:yes" or "fork:only" - Include or only show forks
  * "archived:yes" or "archived:only" - Include or only show archived repos
  * "visibility:public" or "visibility:private" - Filter by visibility

- File filters:
  * "file:\.js$" - Files with .js extension
  * "file:internal/" - Files in internal directory
  * "-file:test" - Exclude test files
  * "file:has.content(Copyright)" - Files containing "Copyright"
  * "file:has.contributor([email protected])" - Files with specific contributor

- Content filters:
  * "content:\"exact string\"" - Search for exact string
  * "-content:\"unwanted\"" - Exclude files with unwanted content
  * "case:yes" - Case-sensitive search

- Type filters:
  * "type:symbol" - Search for symbols (functions, classes, etc.)
  * "type:file" - Search file content only
  * "type:path" - Search filenames only
  * "type:diff" - Search code changes
  * "type:commit" - Search commit messages

- Commit/diff search:
  * "after:\"1 month ago\"" - Commits after date
  * "before:\"2023-01-01\"" - Commits before date
  * "author:name" - Commits by author
  * "message:\"fix bug\"" - Commits with message

- Result selection:
  * "select:repo" - Show only repository names
  * "select:file" - Show only file paths
  * "select:content" - Show only matching content
  * "select:symbol" - Show only matching symbols

- Result control:
  * "count:100" - Return up to 100 results
  * "count:all" - Return all results
  * "timeout:30s" - Set search timeout

EXAMPLES:
- "file:.go context.WithTimeout" - Find Go code using context.WithTimeout
- "lang:typescript useState type:symbol" - Find TypeScript React useState hooks
- "repo:^github\.com/kubernetes/kubernetes$ pod list type:file" - Find Kubernetes files related to pod listing
- "repo:sourcegraph/sourcegraph$ after:\"3 months ago\" type:diff database" - Recent changes to database code
- "file:Dockerfile (alpine OR ubuntu) -content:alpine:latest" - Dockerfiles with specific base images
- "repo:has.path(\.py) file:requirements.txt tensorflow" - Python projects using TensorFlow

BOOLEAN OPERATORS:
- "term1 AND term2" - Results containing both terms
- "term1 OR term2" - Results containing either term
- "term1 NOT term2" - Results with term1 but not term2
- "term1 and (term2 or term3)" - Grouping with parentheses

LIMITATIONS:
- Only searches public repositories
- Rate limits may apply
- Complex queries may take longer to execute
- Maximum of 20 results per query

TIPS:
- Use specific file extensions to narrow results
- Add repo: filters for more targeted searches
- Use type:symbol to find function/method definitions
- Use type:file to find relevant files
- For more details on query syntax, visit: https://docs.sourcegraph.com/code_search/queries`
)

type SourcegraphParams struct {
	Query   string `json:"query"`
	Count   int    `json:"count,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

type SourcegraphPermissionsParams struct {
	Query   string `json:"query"`
	Count   int    `json:"count,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
}

type sourcegraphTool struct {
	client *http.Client
}

func NewSourcegraphTool() BaseTool {
	return &sourcegraphTool{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (t *sourcegraphTool) Info() ToolInfo {
	return ToolInfo{
		Name:        SourcegraphToolName,
		Description: sourcegraphToolDescription,
		Parameters: map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The Sourcegraph search query",
			},
			"count": map[string]any{
				"type":        "number",
				"description": "Optional number of results to return (default: 10, max: 20)",
			},
			"timeout": map[string]any{
				"type":        "number",
				"description": "Optional timeout in seconds (max 120)",
			},
		},
		Required: []string{"query"},
	}
}

func (t *sourcegraphTool) Run(ctx context.Context, call ToolCall) (ToolResponse, error) {
	var params SourcegraphParams
	if err := json.Unmarshal([]byte(call.Input), &params); err != nil {
		return NewTextErrorResponse("Failed to parse sourcegraph parameters: " + err.Error()), nil
	}

	if params.Query == "" {
		return NewTextErrorResponse("Query parameter is required"), nil
	}

	// Set default count if not specified
	if params.Count <= 0 {
		params.Count = 10
	} else if params.Count > 20 {
		params.Count = 20 // Limit to 20 results
	}

	client := t.client
	if params.Timeout > 0 {
		maxTimeout := 120 // 2 minutes
		if params.Timeout > maxTimeout {
			params.Timeout = maxTimeout
		}
		client = &http.Client{
			Timeout: time.Duration(params.Timeout) * time.Second,
		}
	}

	// GraphQL query for Sourcegraph search
	// Create a properly escaped JSON structure
	type graphqlRequest struct {
		Query     string `json:"query"`
		Variables struct {
			Query string `json:"query"`
		} `json:"variables"`
	}

	request := graphqlRequest{
		Query: "query Search($query: String!) { search(query: $query, version: V2, patternType: standard ) { results { matchCount, limitHit, resultCount, approximateResultCount, missing { name }, timedout { name }, indexUnavailable, results { __typename, ... on FileMatch { repository { name }, file { path, url, content }, lineMatches { preview, lineNumber, offsetAndLengths } } } } } }",
	}
	request.Variables.Query = params.Query

	// Marshal to JSON to ensure proper escaping
	graphqlQueryBytes, err := json.Marshal(request)
	if err != nil {
		return NewTextErrorResponse("Failed to create GraphQL request: " + err.Error()), nil
	}
	graphqlQuery := string(graphqlQueryBytes)

	// Create request to Sourcegraph API
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://sourcegraph.com/.api/graphql",
		bytes.NewBuffer([]byte(graphqlQuery)),
	)
	if err != nil {
		return NewTextErrorResponse("Failed to create request: " + err.Error()), nil
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "termai/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return NewTextErrorResponse("Failed to execute request: " + err.Error()), nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// log the error response
		body, _ := io.ReadAll(resp.Body)
		if len(body) > 0 {
			return NewTextErrorResponse(fmt.Sprintf("Request failed with status code: %d, response: %s", resp.StatusCode, string(body))), nil
		}

		return NewTextErrorResponse(fmt.Sprintf("Request failed with status code: %d", resp.StatusCode)), nil
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewTextErrorResponse("Failed to read response body: " + err.Error()), nil
	}

	// Parse the GraphQL response
	var result map[string]any
	if err = json.Unmarshal(body, &result); err != nil {
		return NewTextErrorResponse("Failed to parse response: " + err.Error()), nil
	}

	// Format the results in a readable way
	formattedResults, err := formatSourcegraphResults(result)
	if err != nil {
		return NewTextErrorResponse("Failed to format results: " + err.Error()), nil
	}

	return NewTextResponse(formattedResults), nil
}

func formatSourcegraphResults(result map[string]any) (string, error) {
	var buffer strings.Builder

	// Check for errors in the GraphQL response
	if errors, ok := result["errors"].([]any); ok && len(errors) > 0 {
		buffer.WriteString("## Sourcegraph API Error\n\n")
		for _, err := range errors {
			if errMap, ok := err.(map[string]any); ok {
				if message, ok := errMap["message"].(string); ok {
					buffer.WriteString(fmt.Sprintf("- %s\n", message))
				}
			}
		}
		return buffer.String(), nil
	}

	// Extract data from the response
	data, ok := result["data"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("invalid response format: missing data field")
	}

	search, ok := data["search"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("invalid response format: missing search field")
	}

	searchResults, ok := search["results"].(map[string]any)
	if !ok {
		return "", fmt.Errorf("invalid response format: missing results field")
	}

	// Write search metadata
	matchCount, _ := searchResults["matchCount"].(float64)
	resultCount, _ := searchResults["resultCount"].(float64)
	limitHit, _ := searchResults["limitHit"].(bool)

	buffer.WriteString("# Sourcegraph Search Results\n\n")
	buffer.WriteString(fmt.Sprintf("Found %d matches across %d results\n", int(matchCount), int(resultCount)))

	if limitHit {
		buffer.WriteString("(Result limit reached, try a more specific query)\n")
	}

	buffer.WriteString("\n")

	// Process results
	results, ok := searchResults["results"].([]any)
	if !ok || len(results) == 0 {
		buffer.WriteString("No results found. Try a different query.\n")
		return buffer.String(), nil
	}

	// Limit to 10 results
	maxResults := 10
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	// Process each result
	for i, res := range results {
		fileMatch, ok := res.(map[string]any)
		if !ok {
			continue
		}

		// Skip non-FileMatch results
		typeName, _ := fileMatch["__typename"].(string)
		if typeName != "FileMatch" {
			continue
		}

		// Extract repository and file information
		repo, _ := fileMatch["repository"].(map[string]any)
		file, _ := fileMatch["file"].(map[string]any)
		lineMatches, _ := fileMatch["lineMatches"].([]any)

		if repo == nil || file == nil {
			continue
		}

		repoName, _ := repo["name"].(string)
		filePath, _ := file["path"].(string)
		fileURL, _ := file["url"].(string)
		fileContent, _ := file["content"].(string)

		buffer.WriteString(fmt.Sprintf("## Result %d: %s/%s\n\n", i+1, repoName, filePath))

		if fileURL != "" {
			buffer.WriteString(fmt.Sprintf("URL: %s\n\n", fileURL))
		}

		// Show line matches with context
		if len(lineMatches) > 0 {
			for _, lm := range lineMatches {
				lineMatch, ok := lm.(map[string]any)
				if !ok {
					continue
				}

				lineNumber, _ := lineMatch["lineNumber"].(float64)
				preview, _ := lineMatch["preview"].(string)

				// Extract context from file content if available
				if fileContent != "" {
					lines := strings.Split(fileContent, "\n")

					buffer.WriteString("```\n")

					// Display context before the match (up to 10 lines)
					contextBefore := 10
					startLine := max(1, int(lineNumber)-contextBefore)

					for j := startLine - 1; j < int(lineNumber)-1 && j < len(lines); j++ {
						if j >= 0 {
							buffer.WriteString(fmt.Sprintf("%d| %s\n", j+1, lines[j]))
						}
					}

					// Display the matching line (highlighted)
					buffer.WriteString(fmt.Sprintf("%d|  %s\n", int(lineNumber), preview))

					// Display context after the match (up to 10 lines)
					contextAfter := 10
					endLine := int(lineNumber) + contextAfter

					for j := int(lineNumber); j < endLine && j < len(lines); j++ {
						if j < len(lines) {
							buffer.WriteString(fmt.Sprintf("%d| %s\n", j+1, lines[j]))
						}
					}

					buffer.WriteString("```\n\n")
				} else {
					// If file content is not available, just show the preview
					buffer.WriteString("```\n")
					buffer.WriteString(fmt.Sprintf("%d| %s\n", int(lineNumber), preview))
					buffer.WriteString("```\n\n")
				}
			}
		}
	}

	return buffer.String(), nil
}
