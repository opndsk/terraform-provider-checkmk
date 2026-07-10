package hosts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/terraform-provider-checkmk/internal/client"
	"github.com/terraform-provider-checkmk/internal/common"
)

func TestHostResourceUpdateFolderMoveWithoutAttributes(t *testing.T) {
	var requests []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/check_mk/api/1.0/objects/host_config/example.com/actions/move/invoke":
			if got := r.Header.Get("If-Match"); got != "*" {
				t.Fatalf("expected wildcard If-Match, got %q", got)
			}

			var req client.HostMoveRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("failed to decode move request: %v", err)
			}
			if req.TargetFolder != "~target~folder" {
				t.Fatalf("expected target folder %q, got %q", "~target~folder", req.TargetFolder)
			}

			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "example.com",
				"title": "example.com",
				"extensions": {
					"folder": "/target/folder",
					"attributes": {
						"alias": "Existing Alias",
						"ipaddress": "127.0.0.1"
					}
				}
			}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	apiClient := &client.Client{
		HTTPClient: server.Client(),
		BaseURL:    mustParseURL(t, server.URL),
	}

	r := &HostResource{
		providerData: &common.ProviderData{
			Client:   apiClient,
			Activate: "manual",
		},
	}

	schemaResp := resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)

	plan := HostResourceModel{
		ID:                    types.StringValue("example.com"),
		HostName:              types.StringValue("example.com"),
		Folder:                types.StringValue("/target/folder"),
		Attributes:            types.MapNull(types.StringType),
		Activate:              types.StringNull(),
		ForceForeignChanges:   types.BoolNull(),
		StrictResourceLocking: types.BoolNull(),
	}
	state := HostResourceModel{
		ID:                    types.StringValue("example.com"),
		HostName:              types.StringValue("example.com"),
		Folder:                types.StringValue("/source/folder"),
		Attributes:            mustStringMap(t, map[string]string{"alias": "Existing Alias", "ipaddress": "127.0.0.1"}),
		Activate:              types.StringNull(),
		ForceForeignChanges:   types.BoolNull(),
		StrictResourceLocking: types.BoolNull(),
	}

	planData := tfsdk.Plan{Schema: schemaResp.Schema}
	if diags := planData.Set(context.Background(), plan); diags.HasError() {
		t.Fatalf("failed to build plan: %v", diags)
	}

	stateData := tfsdk.State{Schema: schemaResp.Schema}
	if diags := stateData.Set(context.Background(), state); diags.HasError() {
		t.Fatalf("failed to build state: %v", diags)
	}

	req := resource.UpdateRequest{
		Plan:  planData,
		State: stateData,
	}
	resp := resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}

	r.Update(context.Background(), req, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update returned diagnostics: %v", resp.Diagnostics)
	}

	if len(requests) != 1 {
		t.Fatalf("expected exactly one request, got %d: %v", len(requests), requests)
	}
}

func mustStringMap(t *testing.T, values map[string]string) types.Map {
	t.Helper()

	elements := make(map[string]types.String, len(values))
	for k, v := range values {
		elements[k] = types.StringValue(v)
	}

	m, diags := types.MapValueFrom(context.Background(), types.StringType, elements)
	if diags.HasError() {
		t.Fatalf("failed to build string map: %v", diags)
	}

	return m
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("failed to parse url %q: %v", raw, err)
	}

	return parsed
}
