package main

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPresignedURLExpiration(t *testing.T) {
	// Test SigV4 Expired
	{
		req := httptest.NewRequest("PUT", "/bucket/object", nil)
		q := req.URL.Query()
		// Set date to 10 seconds ago, expires in 5 seconds
		dateStr := time.Now().UTC().Add(-10 * time.Second).Format(iso8601Format)
		q.Set("X-Amz-Date", dateStr)
		q.Set("X-Amz-Expires", "5")
		req.URL.RawQuery = q.Encode()

		rr := httptest.NewRecorder()
		s3Handler(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", rr.Code)
		}

		var resp APIErrorResponse
		if err := xml.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode XML response: %v", err)
		}

		if resp.Code != "RequestExpired" {
			t.Errorf("expected error code RequestExpired, got %s", resp.Code)
		}
	}

	// Test SigV4 Valid
	{
		req := httptest.NewRequest("PUT", "/bucket/object", nil)
		q := req.URL.Query()
		// Set date to now, expires in 60 seconds
		dateStr := time.Now().UTC().Format(iso8601Format)
		q.Set("X-Amz-Date", dateStr)
		q.Set("X-Amz-Expires", "60")
		req.URL.RawQuery = q.Encode()

		rr := httptest.NewRecorder()
		s3Handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	}

	// Test SigV2 Expired
	{
		req := httptest.NewRequest("PUT", "/bucket/object", nil)
		q := req.URL.Query()
		// Set expires to 5 seconds ago
		expiresVal := time.Now().UTC().Add(-5 * time.Second).Unix()
		q.Set("Expires", fmt.Sprintf("%d", expiresVal))
		req.URL.RawQuery = q.Encode()

		rr := httptest.NewRecorder()
		s3Handler(rr, req)

		if rr.Code != http.StatusForbidden {
			t.Errorf("expected status 403, got %d", rr.Code)
		}

		var resp APIErrorResponse
		if err := xml.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode XML response: %v", err)
		}

		if resp.Code != "RequestExpired" {
			t.Errorf("expected error code RequestExpired, got %s", resp.Code)
		}
	}

	// Test SigV2 Valid
	{
		req := httptest.NewRequest("PUT", "/bucket/object", nil)
		q := req.URL.Query()
		// Set expires to 60 seconds from now
		expiresVal := time.Now().UTC().Add(60 * time.Second).Unix()
		q.Set("Expires", fmt.Sprintf("%d", expiresVal))
		req.URL.RawQuery = q.Encode()

		rr := httptest.NewRecorder()
		s3Handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	}
}