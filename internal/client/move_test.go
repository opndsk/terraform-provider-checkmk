package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMoveHost(t *testing.T) {
	tests := []struct {
		name           string
		targetFolder   string
		etag           string
		responseStatus int
		responseBody   string
		expectError    bool
		errorContains  string
		expectedFolder string
	}{
		{
			name:           "successful move with strict etag",
			targetFolder:   "/servers/linux",
			etag:           `"valid-etag"`,
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": "example.com",
				"title": "example.com",
				"extensions": {
					"folder": "/servers/linux",
					"attributes": {
						"alias": "Moved Host"
					}
				}
			}`,
			expectedFolder: "~servers~linux",
		},
		{
			name:           "wildcard etag for non-strict locking",
			targetFolder:   "~servers~windows",
			etag:           "",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": "example.com",
				"title": "example.com",
				"extensions": {
					"folder": "/servers/windows",
					"attributes": {}
				}
			}`,
			expectedFolder: "~servers~windows",
		},
		{
			name:           "precondition failed",
			targetFolder:   "/servers/linux",
			etag:           `"invalid-etag"`,
			responseStatus: http.StatusPreconditionFailed,
			responseBody: `{
				"title": "Precondition Failed",
				"status": 412,
				"detail": "ETag mismatch"
			}`,
			expectError:    true,
			errorContains:  "modified outside of Terraform",
			expectedFolder: "~servers~linux",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/check_mk/api/1.0/objects/host_config/example.com/actions/move/invoke" {
					t.Errorf("Unexpected path: %s", r.URL.Path)
				}

				expectedIfMatch := tt.etag
				if expectedIfMatch == "" {
					expectedIfMatch = "*"
				}
				if ifMatch := r.Header.Get("If-Match"); ifMatch != expectedIfMatch {
					t.Errorf("Expected If-Match header %s, got %s", expectedIfMatch, ifMatch)
				}

				var req HostMoveRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatalf("Failed to decode request body: %v", err)
				}
				if req.TargetFolder != tt.expectedFolder {
					t.Errorf("Expected target_folder %s, got %s", tt.expectedFolder, req.TargetFolder)
				}

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := &Client{
				HTTPClient: server.Client(),
				BaseURL:    mustParseURL(server.URL),
			}

			host, err := client.MoveHost(context.Background(), "example.com", tt.targetFolder, tt.etag)
			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Fatalf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if host == nil {
				t.Fatal("Expected host but got nil")
			}
			if host.Extensions.Folder == "" {
				t.Fatal("Expected moved host folder in response")
			}
		})
	}
}

func TestMoveFolder(t *testing.T) {
	tests := []struct {
		name                string
		folderPath          string
		destination         string
		etag                string
		responseStatus      int
		responseBody        string
		expectError         bool
		errorContains       string
		expectedPath        string
		expectedDestination string
	}{
		{
			name:           "successful move with strict etag",
			folderPath:     "/source/child",
			destination:    "/target",
			etag:           `"valid-etag"`,
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": "/target/child",
				"title": "Child",
				"extensions": {
					"path": "/target/child",
					"attributes": {}
				}
			}`,
			expectedPath:        "/check_mk/api/1.0/objects/folder_config/~source~child/actions/move/invoke",
			expectedDestination: "~target",
		},
		{
			name:           "wildcard etag for non-strict locking",
			folderPath:     "~source~child",
			destination:    "~target",
			etag:           "",
			responseStatus: http.StatusOK,
			responseBody: `{
				"id": "/target/child",
				"title": "Child",
				"extensions": {
					"path": "/target/child",
					"attributes": {}
				}
			}`,
			expectedPath:        "/check_mk/api/1.0/objects/folder_config/~source~child/actions/move/invoke",
			expectedDestination: "~target",
		},
		{
			name:           "precondition failed",
			folderPath:     "/source/child",
			destination:    "/target",
			etag:           `"invalid-etag"`,
			responseStatus: http.StatusPreconditionFailed,
			responseBody: `{
				"title": "Precondition Failed",
				"status": 412,
				"detail": "ETag mismatch"
			}`,
			expectError:         true,
			errorContains:       "modified outside of Terraform",
			expectedPath:        "/check_mk/api/1.0/objects/folder_config/~source~child/actions/move/invoke",
			expectedDestination: "~target",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.URL.Path != tt.expectedPath {
					t.Errorf("Expected path %s, got %s", tt.expectedPath, r.URL.Path)
				}

				expectedIfMatch := tt.etag
				if expectedIfMatch == "" {
					expectedIfMatch = "*"
				}
				if ifMatch := r.Header.Get("If-Match"); ifMatch != expectedIfMatch {
					t.Errorf("Expected If-Match header %s, got %s", expectedIfMatch, ifMatch)
				}

				var req FolderMoveRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Fatalf("Failed to decode request body: %v", err)
				}
				if req.Destination != tt.expectedDestination {
					t.Errorf("Expected destination %s, got %s", tt.expectedDestination, req.Destination)
				}

				w.WriteHeader(tt.responseStatus)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			client := &Client{
				HTTPClient: server.Client(),
				BaseURL:    mustParseURL(server.URL),
			}

			folder, err := client.MoveFolder(context.Background(), tt.folderPath, tt.destination, tt.etag)
			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Fatalf("Expected error containing %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if folder == nil {
				t.Fatal("Expected folder but got nil")
			}
			if folder.Extensions.Path == "" {
				t.Fatal("Expected moved folder path in response")
			}
		})
	}
}
