package scraper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// searchBooks queries Open Library and Google Books in parallel.
func searchBooks(ctx context.Context, query string) ([]SearchResult, error) {
	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		results []SearchResult
		errs    []error
	)

	wg.Add(2)

	go func() {
		defer wg.Done()
		r, err := searchOpenLibrary(ctx, query)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			errs = append(errs, fmt.Errorf("openlibrary: %w", err))
			return
		}
		results = append(results, r...)
	}()

	go func() {
		defer wg.Done()
		r, err := searchGoogleBooks(ctx, query)
		mu.Lock()
		defer mu.Unlock()
		if err != nil {
			errs = append(errs, fmt.Errorf("googlebooks: %w", err))
			return
		}
		results = append(results, r...)
	}()

	wg.Wait()

	// Deduplicate by ISBN
	results = deduplicateResults(results)

	// If both failed, return error
	if len(results) == 0 && len(errs) > 0 {
		return nil, fmt.Errorf("all search providers failed: %v", errs)
	}

	return results, nil
}

// searchOpenLibrary queries the Open Library search API.
func searchOpenLibrary(ctx context.Context, query string) ([]SearchResult, error) {
	u := fmt.Sprintf("https://openlibrary.org/search.json?q=%s&limit=10", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var data struct {
		Docs []struct {
			Title     string   `json:"title"`
			Author    []string `json:"author_name"`
			ISBN      []string `json:"isbn"`
			Publisher []string `json:"publisher"`
			CoverI    int      `json:"cover_i"`
		} `json:"docs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var results []SearchResult
	for _, doc := range data.Docs {
		r := SearchResult{
			Title:  doc.Title,
			Source: "openlibrary",
		}
		if len(doc.Author) > 0 {
			r.Author = doc.Author[0]
		}
		if len(doc.ISBN) > 0 {
			r.ISBN = doc.ISBN[0]
		}
		if len(doc.Publisher) > 0 {
			r.Publisher = doc.Publisher[0]
		}
		if doc.CoverI > 0 {
			r.CoverURL = fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-L.jpg", doc.CoverI)
		}
		results = append(results, r)
	}

	return results, nil
}

// searchGoogleBooks queries the Google Books API.
func searchGoogleBooks(ctx context.Context, query string) ([]SearchResult, error) {
	u := fmt.Sprintf("https://www.googleapis.com/books/v1/volumes?q=%s&maxResults=10", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var data struct {
		Items []struct {
			VolumeInfo struct {
				Title               string   `json:"title"`
				Authors             []string `json:"authors"`
				Publisher           string   `json:"publisher"`
				IndustryIdentifiers []struct {
					Type       string `json:"type"`
					Identifier string `json:"identifier"`
				} `json:"industryIdentifiers"`
				ImageLinks struct {
					Thumbnail string `json:"thumbnail"`
				} `json:"imageLinks"`
				PreviewLink string `json:"previewLink"`
			} `json:"volumeInfo"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var results []SearchResult
	for _, item := range data.Items {
		r := SearchResult{
			Title:      item.VolumeInfo.Title,
			Publisher:  item.VolumeInfo.Publisher,
			PreviewURL: item.VolumeInfo.PreviewLink,
			Source:     "googlebooks",
		}
		if len(item.VolumeInfo.Authors) > 0 {
			r.Author = item.VolumeInfo.Authors[0]
		}
		for _, id := range item.VolumeInfo.IndustryIdentifiers {
			if id.Type == "ISBN_13" || id.Type == "ISBN_10" {
				r.ISBN = id.Identifier
				break
			}
		}
		if item.VolumeInfo.ImageLinks.Thumbnail != "" {
			r.CoverURL = item.VolumeInfo.ImageLinks.Thumbnail
		}
		results = append(results, r)
	}

	return results, nil
}

// deduplicateResults removes duplicates by ISBN, preferring OpenLibrary.
func deduplicateResults(results []SearchResult) []SearchResult {
	seen := make(map[string]bool)
	var deduped []SearchResult

	for _, r := range results {
		key := r.ISBN
		if key == "" {
			key = r.Title + "|" + r.Author // fallback dedup key
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, r)
	}

	return deduped
}
