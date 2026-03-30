package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is an HTTP client for the VaultDrift API
type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

// NewClient creates a new API client
func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetToken sets the authentication token
func (c *Client) SetToken(token string) {
	c.Token = token
}

// request makes an HTTP request to the API
func (c *Client) request(method, path string, body interface{}, query map[string]string) (*http.Response, error) {
	// Build URL
	u, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return nil, err
	}

	// Add query parameters
	if query != nil {
		q := u.Query()
		for key, value := range query {
			q.Set(key, value)
		}
		u.RawQuery = q.Encode()
	}

	// Prepare body
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(data)
	}

	// Create request
	req, err := http.NewRequest(method, u.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	return c.HTTPClient.Do(req)
}

// doRequest performs the request and decodes the response
func (c *Client) doRequest(method, path string, body interface{}, query map[string]string, result interface{}) error {
	resp, err := c.request(method, path, body, query)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(data))
	}

	if result != nil {
		return json.NewDecoder(resp.Body).Decode(result)
	}

	return nil
}

// AuthResponse represents an authentication response
type AuthResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
}

// Login authenticates a user
func (c *Client) Login(username, password string) (*AuthResponse, error) {
	body := map[string]string{
		"username": username,
		"password": password,
	}

	var result struct {
		Data AuthResponse `json:"data"`
	}

	if err := c.doRequest("POST", "/api/v1/auth/login", body, nil, &result); err != nil {
		return nil, err
	}

	c.Token = result.Data.Token
	return &result.Data, nil
}

// Logout invalidates the current token
func (c *Client) Logout() error {
	return c.doRequest("POST", "/api/v1/auth/logout", nil, nil, nil)
}

// File represents a file or folder
type File struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"` // "file" or "folder"
	Size      int64  `json:"size"`
	MimeType  string `json:"mime_type"`
	ParentID  *string `json:"parent_id"`
	UserID    string `json:"user_id"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// ListFiles lists files in a folder
func (c *Client) ListFiles(parentID string, limit, offset int) ([]File, error) {
	query := map[string]string{
		"limit":  fmt.Sprintf("%d", limit),
		"offset": fmt.Sprintf("%d", offset),
	}
	if parentID != "" {
		query["parent_id"] = parentID
	}

	var result struct {
		Data struct {
			Files []File `json:"files"`
		} `json:"data"`
	}

	if err := c.doRequest("GET", "/api/v1/files", nil, query, &result); err != nil {
		return nil, err
	}

	return result.Data.Files, nil
}

// GetFile gets file details
func (c *Client) GetFile(fileID string) (*File, error) {
	var result struct {
		Data File `json:"data"`
	}

	if err := c.doRequest("GET", "/api/v1/files/"+fileID, nil, nil, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// CreateFolder creates a new folder
func (c *Client) CreateFolder(name, parentID string) (*File, error) {
	body := map[string]string{
		"name": name,
	}
	if parentID != "" {
		body["parent_id"] = parentID
	}

	var result struct {
		Data File `json:"data"`
	}

	if err := c.doRequest("POST", "/api/v1/folders", body, nil, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// DeleteFile deletes a file or folder
func (c *Client) DeleteFile(fileID string) error {
	return c.doRequest("DELETE", "/api/v1/files/"+fileID, nil, nil, nil)
}

// DeleteFolder deletes a folder
func (c *Client) DeleteFolder(folderID string) error {
	return c.doRequest("DELETE", "/api/v1/folders/"+folderID, nil, nil, nil)
}

// RenameFile renames a file or folder
func (c *Client) RenameFile(fileID, newName string) error {
	body := map[string]string{
		"name": newName,
	}
	return c.doRequest("PUT", "/api/v1/files/"+fileID, body, nil, nil)
}

// MoveFile moves a file or folder
func (c *Client) MoveFile(fileID, newParentID string) error {
	body := map[string]string{
		"parent_id": newParentID,
	}
	return c.doRequest("PUT", "/api/v1/files/"+fileID, body, nil, nil)
}

// SearchFiles searches for files
func (c *Client) SearchFiles(query string, limit int) ([]File, error) {
	q := map[string]string{
		"q":     query,
		"limit": fmt.Sprintf("%d", limit),
	}

	var result struct {
		Data struct {
			Files []File `json:"files"`
		} `json:"data"`
	}

	if err := c.doRequest("GET", "/api/v1/files/search", nil, q, &result); err != nil {
		return nil, err
	}

	return result.Data.Files, nil
}

// UploadURLResponse represents a response with upload URL
type UploadURLResponse struct {
	UploadURL string `json:"upload_url"`
	FileID    string `json:"file_id"`
}

// GetUploadURL gets a presigned upload URL
func (c *Client) GetUploadURL(parentID, name, mimeType string, size int64) (*UploadURLResponse, error) {
	body := map[string]interface{}{
		"name":      name,
		"mime_type": mimeType,
		"size":      size,
	}
	if parentID != "" {
		body["parent_id"] = parentID
	}

	var result struct {
		Data UploadURLResponse `json:"data"`
	}

	if err := c.doRequest("POST", "/api/v1/uploads", body, nil, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// DownloadURLResponse represents a response with download URL
type DownloadURLResponse struct {
	DownloadURL string `json:"download_url"`
	Filename    string `json:"filename"`
	ExpiresAt   int64  `json:"expires_at"`
}

// GetDownloadURL gets a presigned download URL
func (c *Client) GetDownloadURL(fileID string) (*DownloadURLResponse, error) {
	var result struct {
		Data DownloadURLResponse `json:"data"`
	}

	if err := c.doRequest("GET", "/api/v1/downloads/"+fileID, nil, nil, &result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

// Share represents a share link
type Share struct {
	ID           string  `json:"id"`
	FileID       string  `json:"file_id"`
	CreatedBy    string  `json:"created_by"`
	ShareType    string  `json:"share_type"`
	Token        *string `json:"token,omitempty"`
	ExpiresAt    *int64  `json:"expires_at,omitempty"`
	MaxDownloads *int    `json:"max_downloads,omitempty"`
	AllowUpload  bool    `json:"allow_upload"`
	PreviewOnly  bool    `json:"preview_only"`
	Permission   string  `json:"permission"`
	IsActive     bool    `json:"is_active"`
	CreatedAt    int64   `json:"created_at"`
}

// CreateShareRequest represents a share creation request
type CreateShareRequest struct {
	ShareType    string  `json:"share_type"`
	SharedWith   *string `json:"shared_with,omitempty"`
	Password     *string `json:"password,omitempty"`
	ExpiresDays  *int    `json:"expires_days,omitempty"`
	MaxDownloads *int    `json:"max_downloads,omitempty"`
	AllowUpload  bool    `json:"allow_upload"`
	PreviewOnly  bool    `json:"preview_only"`
	Permission   string  `json:"permission"`
}

// CreateShare creates a new share
func (c *Client) CreateShare(fileID string, req *CreateShareRequest) (*Share, string, error) {
	var result struct {
		Data struct {
			Share    Share   `json:"share"`
			ShareURL *string `json:"share_url,omitempty"`
		} `json:"data"`
	}

	if err := c.doRequest("POST", "/api/v1/files/"+fileID+"/shares", req, nil, &result); err != nil {
		return nil, "", err
	}

	shareURL := ""
	if result.Data.ShareURL != nil {
		shareURL = *result.Data.ShareURL
	}

	return &result.Data.Share, shareURL, nil
}

// ListShares lists shares for a file
func (c *Client) ListShares(fileID string) ([]Share, error) {
	var result struct {
		Data struct {
			Shares []Share `json:"shares"`
		} `json:"data"`
	}

	if err := c.doRequest("GET", "/api/v1/files/"+fileID+"/shares", nil, nil, &result); err != nil {
		return nil, err
	}

	return result.Data.Shares, nil
}

// RevokeShare revokes a share
func (c *Client) RevokeShare(shareID string) error {
	return c.doRequest("DELETE", "/api/v1/shares/"+shareID, nil, nil, nil)
}
