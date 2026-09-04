package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/elacy/terraform-provider-pfsense/v2/internal/client"
)

func TestApplyNowIsSynchronous(t *testing.T) {
	payload := applyNow(map[string]any{"name": "test"})
	if payload["apply"] != true {
		t.Fatalf("apply = %#v, want true", payload["apply"])
	}
	if payload["async"] != false {
		t.Fatalf("async = %#v, want false", payload["async"])
	}
}

func TestDHCPStaticMappingWaitUntilCreated(t *testing.T) {
	var requests atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := requests.Add(1)
		if attempt < 3 {
			writeEnvelope(w, http.StatusOK, []map[string]any{})
			return
		}
		writeEnvelope(w, http.StatusOK, []map[string]any{{
			"id":        0,
			"parent_id": "opt7",
			"mac":       "aa:bb:cc:dd:ee:ff",
		}})
	}))
	defer srv.Close()

	configuredClient, err := client.New(client.Config{URL: srv.URL, APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	resource := &dhcpStaticMappingResource{client: configuredClient}
	if err := resource.waitUntilCreated(context.Background(), "opt7", "aa:bb:cc:dd:ee:ff"); err != nil {
		t.Fatal(err)
	}
	if got := requests.Load(); got != 3 {
		t.Fatalf("GET request count = %d, want 3", got)
	}
}
