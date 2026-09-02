package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

type dnsResolverSettingsMock struct {
	mu            sync.Mutex
	settings      map[string]any
	updateCount   int
	allApplied    bool
	malformedRead bool
}

func newDNSResolverSettingsMock() *dnsResolverSettingsMock {
	return &dnsResolverSettingsMock{
		settings: map[string]any{
			"enable":                        true,
			"port":                          "53",
			"enablessl":                     false,
			"sslcertref":                    "",
			"tlsport":                       "853",
			"active_interface":              []string{"lan", "opt1"},
			"outgoing_interface":            []string{"wan"},
			"strictout":                     true,
			"system_domain_local_zone_type": "transparent",
			"dnssec":                        true,
			"python":                        false,
			"python_order":                  "pre_validator",
			"python_script":                 "",
			"forwarding":                    false,
			"regdhcp":                       true,
			"regdhcpstatic":                 true,
			"regovpnclients":                false,
			"custom_options":                "",
		},
		allApplied: true,
	}
}

func (m *dnsResolverSettingsMock) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		m.mu.Lock()
		defer m.mu.Unlock()

		switch {
		case r.URL.Path == servicesDNSResolverSettingsPath && r.Method == http.MethodGet:
			if m.malformedRead {
				writeEnvelope(w, http.StatusOK, "not-an-object")
				return
			}
			writeEnvelope(w, http.StatusOK, m.settings)

		case r.URL.Path == servicesDNSResolverSettingsPath && r.Method == http.MethodPatch:
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeEnvelope(w, http.StatusBadRequest, nil)
				return
			}
			applied, _ := body["apply"].(bool)
			m.allApplied = m.allApplied && applied
			delete(body, "apply")
			for key, value := range body {
				m.settings[key] = value
			}
			m.updateCount++
			writeEnvelope(w, http.StatusOK, m.settings)

		default:
			writeEnvelope(w, http.StatusNotFound, nil)
		}
	})
}

func dnsResolverSettingsConfig(url, customOptions string, forwarding bool) string {
	return fmt.Sprintf(`
provider "pfsense" {
  url     = %q
  api_key = "test-key"
}

resource "pfsense_services_dns_resolver_settings" "test" {
  enable                        = true
  port                          = "53"
  enablessl                     = false
  sslcertref                    = ""
  tlsport                       = "853"
  active_interface              = ["lan", "opt1"]
  outgoing_interface            = ["wan"]
  strictout                     = true
  system_domain_local_zone_type = "transparent"
  dnssec                        = true
  python                        = false
  python_order                  = "pre_validator"
  python_script                 = ""
  forwarding                    = %t
  regdhcp                       = true
  regdhcpstatic                 = true
  regovpnclients                = false
  custom_options                = %q
}
`, url, forwarding, customOptions)
}

func TestAccDNSResolverSettingsResource(t *testing.T) {
	mock := newDNSResolverSettingsMock()
	srv := httptest.NewServer(mock.handler())
	defer srv.Close()

	updatedOptions := "server:\\n  private-domain: internal.example.com"
	config := dnsResolverSettingsConfig(srv.URL, updatedOptions, true)

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: dnsResolverSettingsConfig(srv.URL, "server:\\n  private-domain: example.com", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_settings.test", "id", "system"),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_settings.test", "active_interface.#", "2"),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_settings.test", "outgoing_interface.0", "wan"),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_settings.test", "python_order", "pre_validator"),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_settings.test", "forwarding", "false"),
				),
			},
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_settings.test", "custom_options", updatedOptions),
					resource.TestCheckResourceAttr("pfsense_services_dns_resolver_settings.test", "forwarding", "true"),
				),
			},
			{
				PreConfig: func() {
					mock.mu.Lock()
					defer mock.mu.Unlock()
					mock.settings["custom_options"] = "server:\\n  private-domain: manual-drift.example.com"
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: config,
				Check: resource.TestCheckResourceAttr(
					"pfsense_services_dns_resolver_settings.test",
					"custom_options",
					updatedOptions,
				),
			},
			{
				ResourceName:      "pfsense_services_dns_resolver_settings.test",
				ImportState:       true,
				ImportStateId:     "system",
				ImportStateVerify: true,
			},
			{
				ResourceName:  "pfsense_services_dns_resolver_settings.test",
				ImportState:   true,
				ImportStateId: "wrong",
				ExpectError:   regexp.MustCompile("The import ID must be `system`"),
			},
		},
	})

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if mock.updateCount != 3 {
		t.Fatalf("update count = %d, want 3", mock.updateCount)
	}
	if !mock.allApplied {
		t.Fatal("one or more settings updates omitted apply=true")
	}
}

func TestAccDNSResolverSettingsMalformedRead(t *testing.T) {
	mock := newDNSResolverSettingsMock()
	mock.malformedRead = true
	srv := httptest.NewServer(mock.handler())
	defer srv.Close()

	resource.Test(t, resource.TestCase{
		IsUnitTest:               true,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      dnsResolverSettingsConfig(srv.URL, "server:", false),
				ExpectError: regexp.MustCompile("failed to decode DNS Resolver settings"),
			},
		},
	})
}
