package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/bestruirui/bestsub/internal/logger"
	"github.com/bestruirui/bestsub/internal/model"
	"github.com/bestruirui/bestsub/internal/repository"
)

// SubFetcher Subscription content retrieval service
type SubFetcher struct {
	subRepo    repository.SubRepository
	httpClient *http.Client
}

// NewSubFetcher Create a new subscription retrieval service
func NewSubFetcher(subRepo repository.SubRepository) *SubFetcher {
	return &SubFetcher{
		subRepo: subRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
	}
}

// FetchSub Fetch subscription content
func (f *SubFetcher) FetchSub(ctx context.Context, subID int64) (*model.Sub, error) {
	// Get subscription information
	sub, err := f.subRepo.GetByID(ctx, subID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	// Get subscription content
	content, err := f.fetchContent(ctx, sub.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch content: %w", err)
	}

	// Store content to global memory cache
	if err := StoreSubContent(subID, content); err != nil {
		return nil, fmt.Errorf("failed to store content: %w", err)
	}

	// Update last fetch time
	if err := f.subRepo.UpdateLastFetch(ctx, subID); err != nil {
		logger.Error("Failed to update last fetch time: %v", err)
	}

	// Get updated subscription information
	updatedSub, err := f.subRepo.GetByID(ctx, subID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated subscription: %w", err)
	}

	return updatedSub, nil
}

// fetchContent Fetch URL content
func (f *SubFetcher) fetchContent(ctx context.Context, subURL string) (string, error) {
	// Validate URL
	if _, err := url.ParseRequestURI(subURL); err != nil {
		return "", model.ErrInvalidSubURL
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, subURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set request header
	req.Header.Set("User-Agent", "BestSub/1.0")

	// Send request
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected response status: %d", resp.StatusCode)
	}

	// Read response content
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}
