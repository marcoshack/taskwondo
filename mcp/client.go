package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Client is an HTTP client for the Taskwondo REST API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new Taskwondo API client.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// --- Response types ---

type apiResponse struct {
	Data json.RawMessage `json:"data"`
	Meta *listMeta       `json:"meta,omitempty"`
}

type apiError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type listMeta struct {
	Cursor  *string `json:"cursor"`
	HasMore bool    `json:"has_more"`
	Total   int     `json:"total"`
}

type User struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	GlobalRole  string `json:"global_role"`
}

type Project struct {
	ID                string  `json:"id"`
	Name              string  `json:"name"`
	Key               string  `json:"key"`
	Description       *string `json:"description,omitempty"`
	DefaultWorkflowID *string `json:"default_workflow_id,omitempty"`
	ItemCounter       int     `json:"item_counter"`
	MemberCount       int     `json:"member_count,omitempty"`
	OpenCount         int     `json:"open_count,omitempty"`
	InProgressCount   int     `json:"in_progress_count,omitempty"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

type WorkItem struct {
	ID          string   `json:"id"`
	ProjectKey  string   `json:"project_key"`
	ItemNumber  int      `json:"item_number"`
	DisplayID   string   `json:"display_id"`
	Type        string   `json:"type"`
	Title       string   `json:"title"`
	Description *string  `json:"description,omitempty"`
	Status      string   `json:"status"`
	Priority    string   `json:"priority"`
	AssigneeID  *string  `json:"assignee_id,omitempty"`
	ReporterID  string   `json:"reporter_id"`
	QueueID     *string  `json:"queue_id,omitempty"`
	MilestoneID *string  `json:"milestone_id,omitempty"`
	Visibility  string   `json:"visibility"`
	Labels      []string `json:"labels"`
	Complexity  *int     `json:"complexity,omitempty"`
	DueDate     *string  `json:"due_date,omitempty"`
	ResolvedAt  *string  `json:"resolved_at,omitempty"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

type WorkItemList struct {
	Items   []WorkItem `json:"items"`
	Cursor  *string    `json:"cursor"`
	HasMore bool       `json:"has_more"`
	Total   int        `json:"total"`
}

type Comment struct {
	ID         string  `json:"id"`
	AuthorID   *string `json:"author_id,omitempty"`
	Body       string  `json:"body"`
	Visibility string  `json:"visibility"`
	EditCount  int     `json:"edit_count"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

type Workflow struct {
	ID          string           `json:"id"`
	ProjectID   *string          `json:"project_id,omitempty"`
	Name        string           `json:"name"`
	Description *string          `json:"description,omitempty"`
	IsDefault   bool             `json:"is_default"`
	Statuses    []WorkflowStatus `json:"statuses"`
	CreatedAt   string           `json:"created_at"`
	UpdatedAt   string           `json:"updated_at"`
}

type WorkflowStatus struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	Category    string  `json:"category"`
	Position    int     `json:"position"`
	Color       *string `json:"color,omitempty"`
}

// --- API methods ---

func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr apiError
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("%s (HTTP %d)", apiErr.Error.Message, resp.StatusCode)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *Client) GetMe() (*User, error) {
	data, err := c.doRequest("GET", "/api/v1/auth/me", nil)
	if err != nil {
		return nil, err
	}
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	var user User
	if err := json.Unmarshal(resp.Data, &user); err != nil {
		return nil, fmt.Errorf("decode user: %w", err)
	}
	return &user, nil
}

func (c *Client) ListProjects() ([]Project, error) {
	data, err := c.doRequest("GET", "/api/v1/projects", nil)
	if err != nil {
		return nil, err
	}
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	var projects []Project
	if err := json.Unmarshal(resp.Data, &projects); err != nil {
		return nil, fmt.Errorf("decode projects: %w", err)
	}
	return projects, nil
}

func (c *Client) GetProject(key string) (*Project, error) {
	data, err := c.doRequest("GET", "/api/v1/projects/"+url.PathEscape(key), nil)
	if err != nil {
		return nil, err
	}
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	var project Project
	if err := json.Unmarshal(resp.Data, &project); err != nil {
		return nil, fmt.Errorf("decode project: %w", err)
	}
	return &project, nil
}

type ListWorkItemsParams struct {
	Project    string
	Statuses   []string
	Priorities []string
	Types      []string
	Assignee   string
	Search     string
	Limit      int
}

func (c *Client) ListWorkItems(params ListWorkItemsParams) (*WorkItemList, error) {
	q := url.Values{}
	if params.Search != "" {
		q.Set("q", params.Search)
	}
	if len(params.Statuses) > 0 {
		q.Set("status", strings.Join(params.Statuses, ","))
	}
	if len(params.Priorities) > 0 {
		q.Set("priority", strings.Join(params.Priorities, ","))
	}
	if len(params.Types) > 0 {
		q.Set("type", strings.Join(params.Types, ","))
	}
	if params.Assignee != "" {
		q.Set("assignees", params.Assignee)
	}
	if params.Limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", params.Limit))
	}

	path := fmt.Sprintf("/api/v1/projects/%s/items", url.PathEscape(params.Project))
	if len(q) > 0 {
		path += "?" + q.Encode()
	}

	data, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}

	// The list endpoint returns {data: [...], meta: {...}}
	var raw struct {
		Data []WorkItem `json:"data"`
		Meta listMeta   `json:"meta"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode work items: %w", err)
	}

	return &WorkItemList{
		Items:   raw.Data,
		Cursor:  raw.Meta.Cursor,
		HasMore: raw.Meta.HasMore,
		Total:   raw.Meta.Total,
	}, nil
}

func (c *Client) GetWorkItem(projectKey string, itemNumber int) (*WorkItem, error) {
	path := fmt.Sprintf("/api/v1/projects/%s/items/%d", url.PathEscape(projectKey), itemNumber)
	data, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	var item WorkItem
	if err := json.Unmarshal(resp.Data, &item); err != nil {
		return nil, fmt.Errorf("decode work item: %w", err)
	}
	return &item, nil
}

type CreateWorkItemParams struct {
	Project     string   `json:"-"`
	Title       string   `json:"title"`
	Type        string   `json:"type,omitempty"`
	Priority    string   `json:"priority,omitempty"`
	Description *string  `json:"description,omitempty"`
	AssigneeID  *string  `json:"assignee_id,omitempty"`
	Labels      []string `json:"labels,omitempty"`
	DueDate     *string  `json:"due_date,omitempty"`
}

func (c *Client) CreateWorkItem(params CreateWorkItemParams) (*WorkItem, error) {
	path := fmt.Sprintf("/api/v1/projects/%s/items", url.PathEscape(params.Project))
	data, err := c.doRequest("POST", path, params)
	if err != nil {
		return nil, err
	}
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	var item WorkItem
	if err := json.Unmarshal(resp.Data, &item); err != nil {
		return nil, fmt.Errorf("decode work item: %w", err)
	}
	return &item, nil
}

func (c *Client) UpdateWorkItem(projectKey string, itemNumber int, updates map[string]interface{}) (*WorkItem, error) {
	path := fmt.Sprintf("/api/v1/projects/%s/items/%d", url.PathEscape(projectKey), itemNumber)
	data, err := c.doRequest("PATCH", path, updates)
	if err != nil {
		return nil, err
	}
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	var item WorkItem
	if err := json.Unmarshal(resp.Data, &item); err != nil {
		return nil, fmt.Errorf("decode work item: %w", err)
	}
	return &item, nil
}

func (c *Client) ListComments(projectKey string, itemNumber int) ([]Comment, error) {
	path := fmt.Sprintf("/api/v1/projects/%s/items/%d/comments", url.PathEscape(projectKey), itemNumber)
	data, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	var comments []Comment
	if err := json.Unmarshal(resp.Data, &comments); err != nil {
		return nil, fmt.Errorf("decode comments: %w", err)
	}
	return comments, nil
}

type CreateCommentParams struct {
	Body       string `json:"body"`
	Visibility string `json:"visibility,omitempty"`
}

func (c *Client) CreateComment(projectKey string, itemNumber int, params CreateCommentParams) (*Comment, error) {
	path := fmt.Sprintf("/api/v1/projects/%s/items/%d/comments", url.PathEscape(projectKey), itemNumber)
	data, err := c.doRequest("POST", path, params)
	if err != nil {
		return nil, err
	}
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	var comment Comment
	if err := json.Unmarshal(resp.Data, &comment); err != nil {
		return nil, fmt.Errorf("decode comment: %w", err)
	}
	return &comment, nil
}

func (c *Client) ListWorkflows() ([]Workflow, error) {
	data, err := c.doRequest("GET", "/api/v1/workflows", nil)
	if err != nil {
		return nil, err
	}
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	var workflows []Workflow
	if err := json.Unmarshal(resp.Data, &workflows); err != nil {
		return nil, fmt.Errorf("decode workflows: %w", err)
	}
	return workflows, nil
}

type Attachment struct {
	ID          string `json:"id"`
	UploaderID  string `json:"uploader_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	Comment     string `json:"comment"`
	DownloadURL string `json:"download_url"`
	CreatedAt   string `json:"created_at"`
}

func (c *Client) UploadAttachment(projectKey string, itemNumber int, filePath, comment string) (*Attachment, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return nil, fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, f); err != nil {
		return nil, fmt.Errorf("copy file: %w", err)
	}

	if comment != "" {
		if err := writer.WriteField("comment", comment); err != nil {
			return nil, fmt.Errorf("write comment field: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	path := fmt.Sprintf("/api/v1/projects/%s/items/%d/attachments", url.PathEscape(projectKey), itemNumber)
	req, err := http.NewRequest("POST", c.baseURL+path, &body)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Use a longer timeout for file uploads
	uploadClient := &http.Client{Timeout: 2 * time.Minute}
	resp, err := uploadClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr apiError
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("%s (HTTP %d)", apiErr.Error.Message, resp.StatusCode)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	var attachment Attachment
	if err := json.Unmarshal(apiResp.Data, &attachment); err != nil {
		return nil, fmt.Errorf("decode attachment: %w", err)
	}
	return &attachment, nil
}

func (c *Client) ListProjectStatuses(projectKey string) ([]WorkflowStatus, error) {
	path := fmt.Sprintf("/api/v1/projects/%s/workflows/statuses", url.PathEscape(projectKey))
	data, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	var resp apiResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	var statuses []WorkflowStatus
	if err := json.Unmarshal(resp.Data, &statuses); err != nil {
		return nil, fmt.Errorf("decode statuses: %w", err)
	}
	return statuses, nil
}
