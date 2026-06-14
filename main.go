package main

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// APIErrorCode type represents S3 error codes
type APIErrorCode int

const (
	ErrNone APIErrorCode = iota
	ErrRequestExpired
	ErrAccessDenied
)

// APIErrorResponse represents the S3 XML error response
type APIErrorResponse struct {
	XMLName   xml.Name `xml:"Error"`
	Code      string   `xml:"Code"`
	Message   string   `xml:"Message"`
	Resource  string   `xml:"Resource"`
	RequestID string   `xml:"RequestId"`
}

// ISO8601 layout used by AWS S3 SigV4
const iso8601Format = "20060102T150405Z"

// checkRequestExpired checks if the presigned request has expired.
func checkRequestExpired(r *http.Request) APIErrorCode {
	// Check SigV4 expiration
	if dateStr := r.URL.Query().Get("X-Amz-Date"); dateStr != "" {
		expiresStr := r.URL.Query().Get("X-Amz-Expires")
		if expiresStr == "" {
			return ErrAccessDenied
		}

		expires, err := strconv.ParseInt(expiresStr, 10, 64)
		if err != nil {
			return ErrAccessDenied
		}

		// Parse X-Amz-Date
		t, err := time.Parse(iso8601Format, dateStr)
		if err != nil {
			return ErrAccessDenied
		}

		// Check if expired (current server time > X-Amz-Date + X-Amz-Expires)
		if t.Add(time.Duration(expires) * time.Second).Before(time.Now().UTC()) {
			return ErrRequestExpired
		}
	} else if expiresStr := r.URL.Query().Get("Expires"); expiresStr != "" {
		// Check SigV2 expiration
		expires, err := strconv.ParseInt(expiresStr, 10, 64)
		if err != nil {
			return ErrAccessDenied
		}

		// Check if expired (current server time > Expires)
		if time.Unix(expires, 0).Before(time.Now().UTC()) {
			return ErrRequestExpired
		}
	}

	return ErrNone
}

func writeErrorResponse(w http.ResponseWriter, errCode APIErrorCode, r *http.Request) {
	var code, message string
	switch errCode {
	case ErrRequestExpired:
		code = "RequestExpired"
		message = "Request has expired"
	case ErrAccessDenied:
		code = "AccessDenied"
		message = "Access Denied"
	default:
		code = "InternalError"
		message = "Internal Server Error"
	}

	resp := APIErrorResponse{
		Code:      code,
		Message:   message,
		Resource:  r.URL.Path,
		RequestID: "1234567890",
	}

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusForbidden)
	xml.NewEncoder(w).Encode(resp)
}

func s3Handler(w http.ResponseWriter, r *http.Request) {
	// Dynamically check expiration per request to prevent cached validation bypass
	if errCode := checkRequestExpired(r); errCode != ErrNone {
		writeErrorResponse(w, errCode, r)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Upload successful"))
}

func main() {
	http.HandleFunc("/", s3Handler)
	fmt.Println("Starting server on :8080...")
	http.ListenAndServe(":8080", nil)
}