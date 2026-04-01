// Package s3client provides a pure Go S3 client with AWS Signature V4.
// This implementation has zero external dependencies.
package s3client

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"hash"
	"io"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"
)

// Client is an S3 API client.
type Client struct {
	httpClient   *http.Client
	endpoint     string
	bucket       string
	region       string
	accessKey    string
	secretKey    string
	usePathStyle bool
}

// Config holds S3 client configuration.
type Config struct {
	Endpoint     string
	Bucket       string
	Region       string
	AccessKey    string
	SecretKey    string
	UsePathStyle bool
}

// NewClient creates a new S3 client.
func NewClient(cfg Config) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		endpoint:     strings.TrimRight(cfg.Endpoint, "/"),
		bucket:       cfg.Bucket,
		region:       cfg.Region,
		accessKey:    cfg.AccessKey,
		secretKey:    cfg.SecretKey,
		usePathStyle: cfg.UsePathStyle,
	}
}

// ObjectInfo holds metadata about an S3 object.
type ObjectInfo struct {
	Key          string
	Size         int64
	LastModified time.Time
	ETag         string
}

// buildURL constructs the URL for an S3 request.
func (c *Client) buildURL(objectKey string) string {
	if c.usePathStyle {
		return fmt.Sprintf("%s/%s/%s", c.endpoint, c.bucket, objectKey)
	}
	// Virtual hosted-style
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", c.bucket, c.region, objectKey)
}

// PutObject uploads an object to S3.
func (c *Client) PutObject(ctx context.Context, objectKey string, data io.Reader, size int64) error {
	url := c.buildURL(objectKey)

	body, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("failed to read data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Length", fmt.Sprintf("%d", size))
	req.Header.Set("Content-Type", "application/octet-stream")

	if err := c.signRequest(req, body); err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("put object failed: %s - %s", resp.Status, string(body))
	}

	return nil
}

// GetObject retrieves an object from S3.
func (c *Client) GetObject(ctx context.Context, objectKey string) (io.ReadCloser, int64, error) {
	url := c.buildURL(objectKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create request: %w", err)
	}

	if err := c.signRequest(req, nil); err != nil {
		return nil, 0, fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, 0, fmt.Errorf("get object failed: %s", resp.Status)
	}

	return resp.Body, resp.ContentLength, nil
}

// DeleteObject removes an object from S3.
func (c *Client) DeleteObject(ctx context.Context, objectKey string) error {
	url := c.buildURL(objectKey)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if err := c.signRequest(req, nil); err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete object failed: %s - %s", resp.Status, string(body))
	}

	return nil
}

// HeadObject checks if an object exists and returns its metadata.
func (c *Client) HeadObject(ctx context.Context, objectKey string) (*ObjectInfo, error) {
	url := c.buildURL(objectKey)

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if err := c.signRequest(req, nil); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("object not found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("head object failed: %s", resp.Status)
	}

	return &ObjectInfo{
		Key:          objectKey,
		Size:         resp.ContentLength,
		ETag:         strings.Trim(resp.Header.Get("ETag"), `"`),
		LastModified: parseTime(resp.Header.Get("Last-Modified")),
	}, nil
}

// HeadBucket checks if the bucket exists and is accessible.
func (c *Client) HeadBucket(ctx context.Context) error {
	url := c.buildURL("")

	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if err := c.signRequest(req, nil); err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusForbidden {
		return fmt.Errorf("head bucket failed: %s", resp.Status)
	}

	return nil
}

// signRequest signs an HTTP request with AWS Signature V4.
func (c *Client) signRequest(req *http.Request, body []byte) error {
	now := time.Now().UTC()
	dateStamp := now.Format("20060102")
	timeStamp := now.Format("20060102T150405Z")

	// Calculate body hash
	var bodyHash string
	if body != nil {
		bodyHash = hex.EncodeToString(sha256Hash(body))
	} else {
		bodyHash = hex.EncodeToString(sha256Hash([]byte("")))
	}

	// Add required headers
	req.Header.Set("Host", req.URL.Host)
	req.Header.Set("X-Amz-Date", timeStamp)
	req.Header.Set("X-Amz-Content-Sha256", bodyHash)

	// Create canonical request
	canonicalRequest := c.createCanonicalRequest(req, bodyHash)

	// Create string to sign
	scope := fmt.Sprintf("%s/%s/s3/aws4_request", dateStamp, c.region)
	stringToSign := fmt.Sprintf("AWS4-HMAC-SHA256\n%s\n%s\n%s",
		timeStamp, scope, hex.EncodeToString(sha256Hash([]byte(canonicalRequest))))

	// Calculate signature
	signingKey := c.getSigningKey(dateStamp)
	signature := hex.EncodeToString(hmacSHA256(signingKey, []byte(stringToSign)))

	// Add authorization header
	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		c.accessKey, scope, c.getSignedHeaders(req), signature)
	req.Header.Set("Authorization", authHeader)

	return nil
}

// createCanonicalRequest creates the canonical request for signing.
func (c *Client) createCanonicalRequest(req *http.Request, bodyHash string) string {
	// HTTP method
	canonicalRequest := req.Method + "\n"

	// URI path (must be URL-encoded)
	canonicalRequest += path.Join("/", req.URL.Path) + "\n"

	// Query string (sorted by key)
	query := req.URL.Query()
	var queryKeys []string
	for k := range query {
		queryKeys = append(queryKeys, k)
	}
	sort.Strings(queryKeys)
	var queryParts []string
	for _, k := range queryKeys {
		for _, v := range query[k] {
			queryParts = append(queryParts, fmt.Sprintf("%s=%s", k, v))
		}
	}
	canonicalRequest += strings.Join(queryParts, "&") + "\n"

	// Headers (sorted by name, lowercase)
	var headerKeys []string
	headers := make(map[string]string)
	for k, v := range req.Header {
		lowerKey := strings.ToLower(k)
		headerKeys = append(headerKeys, lowerKey)
		headers[lowerKey] = strings.Join(v, ",")
	}
	sort.Strings(headerKeys)

	var canonicalHeaders strings.Builder
	for _, k := range headerKeys {
		canonicalHeaders.WriteString(fmt.Sprintf("%s:%s\n", k, headers[k]))
	}
	canonicalRequest += canonicalHeaders.String() + "\n"

	// Signed headers
	canonicalRequest += c.getSignedHeaders(req) + "\n"

	// Body hash
	canonicalRequest += bodyHash

	return canonicalRequest
}

// getSignedHeaders returns the list of signed headers.
func (c *Client) getSignedHeaders(req *http.Request) string {
	var keys []string
	for k := range req.Header {
		keys = append(keys, strings.ToLower(k))
	}
	sort.Strings(keys)
	return strings.Join(keys, ";")
}

// getSigningKey generates the AWS Signature V4 signing key.
func (c *Client) getSigningKey(dateStamp string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+c.secretKey), []byte(dateStamp))
	kRegion := hmacSHA256(kDate, []byte(c.region))
	kService := hmacSHA256(kRegion, []byte("s3"))
	kSigning := hmacSHA256(kService, []byte("aws4_request"))
	return kSigning
}

// Helper functions
func sha256Hash(data []byte) []byte {
	h := sha256.New()
	h.Write(data)
	return h.Sum(nil)
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func parseTime(s string) time.Time {
	t, _ := http.ParseTime(s)
	return t
}

// HashReader wraps a reader and computes a hash.
type HashReader struct {
	io.Reader
	hash hash.Hash
}

// NewHashReader creates a new HashReader.
func NewHashReader(r io.Reader, h hash.Hash) *HashReader {
	return &HashReader{Reader: r, hash: h}
}

func (hr *HashReader) Read(p []byte) (n int, err error) {
	n, err = hr.Reader.Read(p)
	if n > 0 {
		hr.hash.Write(p[:n])
	}
	return n, err
}

// Sum returns the hex-encoded hash.
func (hr *HashReader) Sum() string {
	return hex.EncodeToString(hr.hash.Sum(nil))
}

// ListObjectsV2 lists objects in a bucket with optional prefix.
func (c *Client) ListObjectsV2(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	url := c.buildURL("")

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	query := req.URL.Query()
	query.Set("list-type", "2")
	if prefix != "" {
		query.Set("prefix", prefix)
	}
	req.URL.RawQuery = query.Encode()

	if err := c.signRequest(req, nil); err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list objects failed: %s - %s", resp.Status, string(body))
	}

	return parseListObjectsV2Response(resp.Body)
}

type listObjectsV2Response struct {
	XMLName  xml.Name  `xml:"ListBucketResult"`
	Contents []objInfo `xml:"Contents"`
}

type objInfo struct {
	Key          string `xml:"Key"`
	Size         int64  `xml:"Size"`
	LastModified string `xml:"LastModified"`
	ETag         string `xml:"ETag"`
}

func parseListObjectsV2Response(r io.Reader) ([]ObjectInfo, error) {
	var resp listObjectsV2Response
	if err := xml.NewDecoder(r).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to parse XML response: %w", err)
	}

	objects := make([]ObjectInfo, len(resp.Contents))
	for i, c := range resp.Contents {
		objects[i] = ObjectInfo{
			Key:          c.Key,
			Size:         c.Size,
			LastModified: parseTime(c.LastModified),
			ETag:         strings.Trim(c.ETag, `"`),
		}
	}
	return objects, nil
}
