package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/veriteknik/registry-proxy/internal/models"
)

// RegistryClient handles communication with the upstream registry
type RegistryClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewRegistryClient creates a new registry client
func NewRegistryClient(baseURL string) *RegistryClient {
	return &RegistryClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ListResponse represents the response from the list endpoint
type ListResponse struct {
	Servers  []models.Server `json:"servers"`
	Metadata struct {
		NextCursor string `json:"next_cursor,omitempty"`
		Count      int    `json:"count"`
	} `json:"metadata"`
}

// GetServers fetches the server list from the registry
func (c *RegistryClient) GetServers(ctx context.Context, cursor string, limit int) (*ListResponse, error) {
	url := fmt.Sprintf("%s/v0/servers?limit=%d", c.baseURL, limit)
	if cursor != "" {
		url += fmt.Sprintf("&cursor=%s", cursor)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var listResp ListResponse
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &listResp, nil
}

// GetServerDetail fetches detailed information for a specific server
func (c *RegistryClient) GetServerDetail(ctx context.Context, serverID string) (*models.ServerDetail, error) {
	url := fmt.Sprintf("%s/v0/servers/%s", c.baseURL, serverID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var detail models.ServerDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &detail, nil
}

// GetAllServersWithDetails fetches all servers and enriches them with package details
func (c *RegistryClient) GetAllServersWithDetails(ctx context.Context) ([]models.EnrichedServer, error) {
	var allServers []models.Server
	cursor := ""
	
	// Fetch all servers (paginated)
	for {
		resp, err := c.GetServers(ctx, cursor, 100)
		if err != nil {
			return nil, fmt.Errorf("fetching servers: %w", err)
		}
		
		allServers = append(allServers, resp.Servers...)
		
		if resp.Metadata.NextCursor == "" {
			break
		}
		cursor = resp.Metadata.NextCursor
	}
	
	// Enrich servers with details in parallel
	enriched := make([]models.EnrichedServer, len(allServers))
	var wg sync.WaitGroup
	var mu sync.Mutex
	errs := make([]error, 0)
	
	// Limit concurrent requests
	semaphore := make(chan struct{}, 10)
	
	for i, server := range allServers {
		wg.Add(1)
		go func(idx int, srv models.Server) {
			defer wg.Done()
			
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			detail, err := c.GetServerDetail(ctx, srv.ID)
			if err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("fetching detail for %s: %w", srv.ID, err))
				mu.Unlock()
				
				// Use basic server info without packages on error
				enriched[idx] = models.EnrichedServer{
					Server:   srv,
					Packages: []models.Package{},
				}
				return
			}
			
			enriched[idx] = models.EnrichedServer{
				Server:   srv,
				Packages: detail.Packages,
			}
		}(i, server)
	}
	
	wg.Wait()
	
	if len(errs) > 0 {
		// Log errors but don't fail completely
		fmt.Printf("Encountered %d errors while fetching details\n", len(errs))
	}
	
	return enriched, nil
}