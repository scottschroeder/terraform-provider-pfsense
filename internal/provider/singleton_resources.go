package provider

import (
	"context"

	"github.com/elacy/terraform-provider-pfsense/v2/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// pfsense_system_hostname (singleton)
// ---------------------------------------------------------------------------

type systemHostnameResource struct{ client *client.Client }

var _ resource.Resource = (*systemHostnameResource)(nil)
var _ resource.ResourceWithConfigure = (*systemHostnameResource)(nil)
var _ resource.ResourceWithImportState = (*systemHostnameResource)(nil)

const systemHostnamePath = "/api/v2/system/hostname"

type systemHostnameModel struct {
	ID       types.String `tfsdk:"id"`
	Hostname types.String `tfsdk:"hostname"`
	Domain   types.String `tfsdk:"domain"`
}

func NewSystemHostnameResource() resource.Resource { return &systemHostnameResource{} }

func (r *systemHostnameResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_system_hostname"
}
func (r *systemHostnameResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *systemHostnameResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the system hostname and domain (a singleton).",
		Attributes: map[string]schema.Attribute{
			"id":       computedIDAttribute(),
			"hostname": requiredStringAttribute("The system hostname (without domain)."),
			"domain":   requiredStringAttribute("The system domain."),
		},
	}
}

func (r *systemHostnameResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan systemHostnameModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "hostname", plan.Hostname)
	setString(payload, "domain", plan.Domain)
	applyNow(payload)
	if _, err := r.client.Update(ctx, systemHostnamePath, payload); err != nil {
		resp.Diagnostics.AddError("failed to set hostname", err.Error())
		return
	}
	plan.ID = types.StringValue("system")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *systemHostnameResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state systemHostnameModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	raw, err := r.client.Get(ctx, systemHostnamePath, nil)
	if err != nil {
		resp.Diagnostics.AddError("failed to read hostname", err.Error())
		return
	}
	obj, _ := decodeObject(raw)
	state.ID = types.StringValue("system")
	state.Hostname = strValue(getString(obj, "hostname"))
	state.Domain = strValue(getString(obj, "domain"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *systemHostnameResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan systemHostnameModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setString(payload, "hostname", plan.Hostname)
	setString(payload, "domain", plan.Domain)
	applyNow(payload)
	if _, err := r.client.Update(ctx, systemHostnamePath, payload); err != nil {
		resp.Diagnostics.AddError("failed to update hostname", err.Error())
		return
	}
	plan.ID = types.StringValue("system")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *systemHostnameResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// System hostname cannot be deleted; removal from state is sufficient.
}

// ImportState adopts the settings already on the box. Singletons have a fixed
// identifier, so the import ID is always `system`.
func (r *systemHostnameResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_system_dns (singleton)
// ---------------------------------------------------------------------------

type systemDNSResource struct{ client *client.Client }

var _ resource.Resource = (*systemDNSResource)(nil)
var _ resource.ResourceWithConfigure = (*systemDNSResource)(nil)
var _ resource.ResourceWithImportState = (*systemDNSResource)(nil)

const systemDNSPath = "/api/v2/system/dns"

type systemDNSModel struct {
	ID               types.String `tfsdk:"id"`
	DNSAllowOverride types.Bool   `tfsdk:"dnsallowoverride"`
	DNSLocalhost     types.String `tfsdk:"dnslocalhost"`
	DNSServer        types.List   `tfsdk:"dnsserver"`
}

func NewSystemDNSResource() resource.Resource { return &systemDNSResource{} }

func (r *systemDNSResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_system_dns"
}
func (r *systemDNSResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *systemDNSResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages system DNS settings (a singleton).",
		Attributes: map[string]schema.Attribute{
			"id":               computedIDAttribute(),
			"dnsallowoverride": optionalBoolAttribute("Allow DNS server list to be overridden by DHCP/PPP on WAN."),
			"dnslocalhost":     optionalStringAttribute("Enable the DNS resolver as a local server (local/remote/none)."),
			"dnsserver":        schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: "DNS server IP addresses."},
		},
	}
}

func (r *systemDNSResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan systemDNSModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setBool(payload, "dnsallowoverride", plan.DNSAllowOverride)
	setString(payload, "dnslocalhost", plan.DNSLocalhost)
	setStringList(payload, "dnsserver", plan.DNSServer)
	applyNow(payload)
	if _, err := r.client.Update(ctx, systemDNSPath, payload); err != nil {
		resp.Diagnostics.AddError("failed to set DNS settings", err.Error())
		return
	}
	plan.ID = types.StringValue("system")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *systemDNSResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state systemDNSModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	raw, err := r.client.Get(ctx, systemDNSPath, nil)
	if err != nil {
		resp.Diagnostics.AddError("failed to read DNS settings", err.Error())
		return
	}
	obj, _ := decodeObject(raw)
	state.ID = types.StringValue("system")
	state.DNSAllowOverride = boolValue(getBool(obj, "dnsallowoverride"))
	state.DNSLocalhost = strValue(getString(obj, "dnslocalhost"))
	state.DNSServer = strListValue(ctx, getStringSlice(obj, "dnsserver"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *systemDNSResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan systemDNSModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{}
	setBool(payload, "dnsallowoverride", plan.DNSAllowOverride)
	setString(payload, "dnslocalhost", plan.DNSLocalhost)
	setStringList(payload, "dnsserver", plan.DNSServer)
	applyNow(payload)
	if _, err := r.client.Update(ctx, systemDNSPath, payload); err != nil {
		resp.Diagnostics.AddError("failed to update DNS settings", err.Error())
		return
	}
	plan.ID = types.StringValue("system")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *systemDNSResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// System DNS settings cannot be deleted; removal from state is sufficient.
}

// ImportState adopts the settings already on the box; the import ID is always
// `system`.
func (r *systemDNSResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_system_timezone (singleton)
// ---------------------------------------------------------------------------

type systemTimezoneResource struct{ client *client.Client }

var _ resource.Resource = (*systemTimezoneResource)(nil)
var _ resource.ResourceWithConfigure = (*systemTimezoneResource)(nil)
var _ resource.ResourceWithImportState = (*systemTimezoneResource)(nil)

const systemTimezonePath = "/api/v2/system/timezone"

type systemTimezoneModel struct {
	ID       types.String `tfsdk:"id"`
	Timezone types.String `tfsdk:"timezone"`
}

func NewSystemTimezoneResource() resource.Resource { return &systemTimezoneResource{} }

func (r *systemTimezoneResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_system_timezone"
}
func (r *systemTimezoneResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *systemTimezoneResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the system timezone (a singleton).",
		Attributes: map[string]schema.Attribute{
			"id":       computedIDAttribute(),
			"timezone": requiredStringAttribute("The system timezone (e.g. `Etc/UTC`, `America/Los_Angeles`)."),
		},
	}
}

func (r *systemTimezoneResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan systemTimezoneModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{"timezone": plan.Timezone.ValueString()}
	applyNow(payload)
	if _, err := r.client.Update(ctx, systemTimezonePath, payload); err != nil {
		resp.Diagnostics.AddError("failed to set timezone", err.Error())
		return
	}
	plan.ID = types.StringValue("system")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *systemTimezoneResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state systemTimezoneModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	raw, err := r.client.Get(ctx, systemTimezonePath, nil)
	if err != nil {
		resp.Diagnostics.AddError("failed to read timezone", err.Error())
		return
	}
	obj, _ := decodeObject(raw)
	state.ID = types.StringValue("system")
	state.Timezone = strValue(getString(obj, "timezone"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *systemTimezoneResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan systemTimezoneModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	payload := map[string]any{"timezone": plan.Timezone.ValueString()}
	applyNow(payload)
	if _, err := r.client.Update(ctx, systemTimezonePath, payload); err != nil {
		resp.Diagnostics.AddError("failed to update timezone", err.Error())
		return
	}
	plan.ID = types.StringValue("system")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *systemTimezoneResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// System timezone cannot be deleted; removal from state is sufficient.
}

// ImportState adopts the timezone already on the box; the import ID is always
// `system`.
func (r *systemTimezoneResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_services_dns_resolver_settings (singleton)
// ---------------------------------------------------------------------------

type servicesDNSResolverSettingsResource struct{ client *client.Client }

var _ resource.Resource = (*servicesDNSResolverSettingsResource)(nil)
var _ resource.ResourceWithConfigure = (*servicesDNSResolverSettingsResource)(nil)
var _ resource.ResourceWithImportState = (*servicesDNSResolverSettingsResource)(nil)

const servicesDNSResolverSettingsPath = "/api/v2/services/dns_resolver/settings"

type servicesDNSResolverSettingsModel struct {
	ID                        types.String `tfsdk:"id"`
	Enable                    types.Bool   `tfsdk:"enable"`
	Port                      types.String `tfsdk:"port"`
	EnableSSL                 types.Bool   `tfsdk:"enablessl"`
	SSLCertRef                types.String `tfsdk:"sslcertref"`
	TLSPort                   types.String `tfsdk:"tlsport"`
	ActiveInterface           types.List   `tfsdk:"active_interface"`
	OutgoingInterface         types.List   `tfsdk:"outgoing_interface"`
	StrictOut                 types.Bool   `tfsdk:"strictout"`
	SystemDomainLocalZoneType types.String `tfsdk:"system_domain_local_zone_type"`
	DNSSEC                    types.Bool   `tfsdk:"dnssec"`
	Python                    types.Bool   `tfsdk:"python"`
	PythonOrder               types.String `tfsdk:"python_order"`
	PythonScript              types.String `tfsdk:"python_script"`
	Forwarding                types.Bool   `tfsdk:"forwarding"`
	RegDHCP                   types.Bool   `tfsdk:"regdhcp"`
	RegDHCPStatic             types.Bool   `tfsdk:"regdhcpstatic"`
	RegOpenVPNClients         types.Bool   `tfsdk:"regovpnclients"`
	CustomOptions             types.String `tfsdk:"custom_options"`
}

func NewServicesDNSResolverSettingsResource() resource.Resource {
	return &servicesDNSResolverSettingsResource{}
}

func (r *servicesDNSResolverSettingsResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_services_dns_resolver_settings"
}

func (r *servicesDNSResolverSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}

func (r *servicesDNSResolverSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages DNS Resolver settings (a singleton).",
		Attributes: map[string]schema.Attribute{
			"id":                            computedIDAttribute(),
			"enable":                        optionalBoolAttribute("Enable the DNS Resolver service."),
			"port":                          optionalStringAttribute("DNS Resolver listening port."),
			"enablessl":                     optionalBoolAttribute("Enable DNS over TLS service."),
			"sslcertref":                    requiredStringAttribute("Certificate reference used by DNS over TLS."),
			"tlsport":                       optionalStringAttribute("DNS over TLS listening port."),
			"active_interface":              optionalStringListAttribute("Interfaces on which the DNS Resolver accepts queries."),
			"outgoing_interface":            optionalStringListAttribute("Interfaces used for outgoing DNS queries."),
			"strictout":                     optionalBoolAttribute("Only use the selected outgoing interfaces."),
			"system_domain_local_zone_type": enumAttribute("Local zone type for the system domain.", "deny", "refuse", "static", "transparent", "typetransparent", "redirect", "inform", "inform_deny", "nodefault"),
			"dnssec":                        optionalBoolAttribute("Enable DNSSEC validation."),
			"python":                        optionalBoolAttribute("Enable the Unbound Python module."),
			"python_order":                  enumAttribute("Order in which the Python module runs.", "pre_validator", "post_validator"),
			"python_script":                 optionalStringAttribute("Unbound Python module script."),
			"forwarding":                    optionalBoolAttribute("Enable DNS forwarding mode."),
			"regdhcp":                       optionalBoolAttribute("Register DHCP leases in the DNS Resolver."),
			"regdhcpstatic":                 optionalBoolAttribute("Register DHCP static mappings in the DNS Resolver."),
			"regovpnclients":                optionalBoolAttribute("Register OpenVPN clients in the DNS Resolver."),
			"custom_options":                optionalStringAttribute("Custom Unbound configuration options."),
		},
	}
}

func (r *servicesDNSResolverSettingsResource) payload(m servicesDNSResolverSettingsModel) map[string]any {
	p := map[string]any{}
	setBool(p, "enable", m.Enable)
	setString(p, "port", m.Port)
	setBool(p, "enablessl", m.EnableSSL)
	setString(p, "sslcertref", m.SSLCertRef)
	setString(p, "tlsport", m.TLSPort)
	setStringList(p, "active_interface", m.ActiveInterface)
	setStringList(p, "outgoing_interface", m.OutgoingInterface)
	setBool(p, "strictout", m.StrictOut)
	setString(p, "system_domain_local_zone_type", m.SystemDomainLocalZoneType)
	setBool(p, "dnssec", m.DNSSEC)
	setBool(p, "python", m.Python)
	setString(p, "python_order", m.PythonOrder)
	setString(p, "python_script", m.PythonScript)
	setBool(p, "forwarding", m.Forwarding)
	setBool(p, "regdhcp", m.RegDHCP)
	setBool(p, "regdhcpstatic", m.RegDHCPStatic)
	setBool(p, "regovpnclients", m.RegOpenVPNClients)
	setString(p, "custom_options", m.CustomOptions)
	return p
}

func (r *servicesDNSResolverSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan servicesDNSResolverSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	if _, err := r.client.Update(ctx, servicesDNSResolverSettingsPath, applyNow(r.payload(plan))); err != nil {
		resp.Diagnostics.AddError("failed to set DNS Resolver settings", err.Error())
		return
	}
	plan.ID = types.StringValue("system")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *servicesDNSResolverSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state servicesDNSResolverSettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	raw, err := r.client.Get(ctx, servicesDNSResolverSettingsPath, nil)
	if err != nil {
		resp.Diagnostics.AddError("failed to read DNS Resolver settings", err.Error())
		return
	}
	obj, err := decodeObject(raw)
	if err != nil {
		resp.Diagnostics.AddError("failed to decode DNS Resolver settings", err.Error())
		return
	}
	state.ID = types.StringValue("system")
	state.Enable = boolValue(getBool(obj, "enable"))
	state.Port = strValue(getString(obj, "port"))
	state.EnableSSL = boolValue(getBool(obj, "enablessl"))
	state.SSLCertRef = strValue(getString(obj, "sslcertref"))
	state.TLSPort = strValue(getString(obj, "tlsport"))
	state.ActiveInterface = strListValue(ctx, getStringSlice(obj, "active_interface"))
	state.OutgoingInterface = strListValue(ctx, getStringSlice(obj, "outgoing_interface"))
	state.StrictOut = boolValue(getBool(obj, "strictout"))
	state.SystemDomainLocalZoneType = strValue(getString(obj, "system_domain_local_zone_type"))
	state.DNSSEC = boolValue(getBool(obj, "dnssec"))
	state.Python = boolValue(getBool(obj, "python"))
	state.PythonOrder = strValue(getString(obj, "python_order"))
	state.PythonScript = strValue(getString(obj, "python_script"))
	state.Forwarding = boolValue(getBool(obj, "forwarding"))
	state.RegDHCP = boolValue(getBool(obj, "regdhcp"))
	state.RegDHCPStatic = boolValue(getBool(obj, "regdhcpstatic"))
	state.RegOpenVPNClients = boolValue(getBool(obj, "regovpnclients"))
	state.CustomOptions = strValue(getString(obj, "custom_options"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *servicesDNSResolverSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan servicesDNSResolverSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	if _, err := r.client.Update(ctx, servicesDNSResolverSettingsPath, applyNow(r.payload(plan))); err != nil {
		resp.Diagnostics.AddError("failed to update DNS Resolver settings", err.Error())
		return
	}
	plan.ID = types.StringValue("system")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *servicesDNSResolverSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// DNS Resolver settings cannot be deleted; removal from state is sufficient.
}

// ImportState adopts the settings already on the box. Singletons have a fixed
// identifier, so the import ID is always `system`.
func (r *servicesDNSResolverSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if req.ID != "system" {
		resp.Diagnostics.AddError("invalid DNS Resolver settings import ID", "The import ID must be `system`.")
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// ---------------------------------------------------------------------------
// pfsense_services_ntp_settings (singleton)
// ---------------------------------------------------------------------------

type servicesNTPSettingsResource struct{ client *client.Client }

var _ resource.Resource = (*servicesNTPSettingsResource)(nil)
var _ resource.ResourceWithConfigure = (*servicesNTPSettingsResource)(nil)
var _ resource.ResourceWithImportState = (*servicesNTPSettingsResource)(nil)

const servicesNTPSettingsPath = "/api/v2/services/ntp/settings"

type servicesNTPSettingsModel struct {
	ID             types.String `tfsdk:"id"`
	Enable         types.Bool   `tfsdk:"enable"`
	Interface      types.List   `tfsdk:"interface"`
	NTPMaxPeers    types.Int64  `tfsdk:"ntpmaxpeers"`
	Orphan         types.Int64  `tfsdk:"orphan"`
	NTPMinPoll     types.String `tfsdk:"ntpminpoll"`
	NTPMaxPoll     types.String `tfsdk:"ntpmaxpoll"`
	DNSResolv      types.String `tfsdk:"dnsresolv"`
	LogPeer        types.Bool   `tfsdk:"logpeer"`
	LogSys         types.Bool   `tfsdk:"logsys"`
	ClockStats     types.Bool   `tfsdk:"clockstats"`
	LoopStats      types.Bool   `tfsdk:"loopstats"`
	PeerStats      types.Bool   `tfsdk:"peerstats"`
	StatsGraph     types.Bool   `tfsdk:"statsgraph"`
	LeapSec        types.String `tfsdk:"leapsec"`
	ServerAuth     types.Bool   `tfsdk:"serverauth"`
	ServerAuthKey  types.String `tfsdk:"serverauthkey"`
	ServerAuthAlgo types.String `tfsdk:"serverauthalgo"`
}

func NewServicesNTPSettingsResource() resource.Resource { return &servicesNTPSettingsResource{} }

func (r *servicesNTPSettingsResource) Metadata(_ context.Context, _ resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "pfsense_services_ntp_settings"
}
func (r *servicesNTPSettingsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = resourceClient(ctx, req, resp)
}
func (r *servicesNTPSettingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages NTP service settings (a singleton).",
		Attributes: map[string]schema.Attribute{
			"id":             computedIDAttribute(),
			"enable":         optionalBoolAttribute("Enable the NTP service."),
			"interface":      schema.ListAttribute{ElementType: types.StringType, Optional: true, Description: "Interfaces to listen on."},
			"ntpmaxpeers":    optionalIntAttribute("Maximum number of peers (4-25)."),
			"orphan":         optionalIntAttribute("Orphan mode stratum (1-15)."),
			"ntpminpoll":     optionalStringAttribute("Minimum poll interval."),
			"ntpmaxpoll":     optionalStringAttribute("Maximum poll interval."),
			"dnsresolv":      optionalStringAttribute("DNS resolution mode (auto/manual)."),
			"logpeer":        optionalBoolAttribute("Log peer messages."),
			"logsys":         optionalBoolAttribute("Log system messages."),
			"clockstats":     optionalBoolAttribute("Log clock statistics."),
			"loopstats":      optionalBoolAttribute("Log loop statistics."),
			"peerstats":      optionalBoolAttribute("Log peer statistics."),
			"statsgraph":     optionalBoolAttribute("Enable statistics graphs."),
			"leapsec":        schema.StringAttribute{Optional: true, Sensitive: true, Description: "Leap second file (base64)."},
			"serverauth":     optionalBoolAttribute("Enable server authentication."),
			"serverauthkey":  schema.StringAttribute{Optional: true, Sensitive: true, Description: "Server authentication key."},
			"serverauthalgo": optionalStringAttribute("Server authentication algorithm (default md5)."),
		},
	}
}

func (r *servicesNTPSettingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan servicesNTPSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	if _, err := r.client.Update(ctx, servicesNTPSettingsPath, applyNow(r.payload(plan))); err != nil {
		resp.Diagnostics.AddError("failed to set NTP settings", err.Error())
		return
	}
	plan.ID = types.StringValue("system")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *servicesNTPSettingsResource) payload(m servicesNTPSettingsModel) map[string]any {
	p := map[string]any{}
	setBool(p, "enable", m.Enable)
	setStringList(p, "interface", m.Interface)
	setInt(p, "ntpmaxpeers", m.NTPMaxPeers)
	setInt(p, "orphan", m.Orphan)
	setString(p, "ntpminpoll", m.NTPMinPoll)
	setString(p, "ntpmaxpoll", m.NTPMaxPoll)
	setString(p, "dnsresolv", m.DNSResolv)
	setBool(p, "logpeer", m.LogPeer)
	setBool(p, "logsys", m.LogSys)
	setBool(p, "clockstats", m.ClockStats)
	setBool(p, "loopstats", m.LoopStats)
	setBool(p, "peerstats", m.PeerStats)
	setBool(p, "statsgraph", m.StatsGraph)
	setString(p, "leapsec", m.LeapSec)
	setBool(p, "serverauth", m.ServerAuth)
	setString(p, "serverauthkey", m.ServerAuthKey)
	setString(p, "serverauthalgo", m.ServerAuthAlgo)
	return p
}

func (r *servicesNTPSettingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state servicesNTPSettingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	raw, err := r.client.Get(ctx, servicesNTPSettingsPath, nil)
	if err != nil {
		resp.Diagnostics.AddError("failed to read NTP settings", err.Error())
		return
	}
	obj, _ := decodeObject(raw)
	state.ID = types.StringValue("system")
	state.Enable = boolValue(getBool(obj, "enable"))
	state.Interface = strListValue(ctx, getStringSlice(obj, "interface"))
	state.NTPMaxPeers = intValue(getInt(obj, "ntpmaxpeers"))
	state.Orphan = intValue(getInt(obj, "orphan"))
	state.NTPMinPoll = strValue(getString(obj, "ntpminpoll"))
	state.NTPMaxPoll = strValue(getString(obj, "ntpmaxpoll"))
	state.DNSResolv = strValue(getString(obj, "dnsresolv"))
	state.LogPeer = boolValue(getBool(obj, "logpeer"))
	state.LogSys = boolValue(getBool(obj, "logsys"))
	state.ClockStats = boolValue(getBool(obj, "clockstats"))
	state.LoopStats = boolValue(getBool(obj, "loopstats"))
	state.PeerStats = boolValue(getBool(obj, "peerstats"))
	state.StatsGraph = boolValue(getBool(obj, "statsgraph"))
	state.ServerAuth = boolValue(getBool(obj, "serverauth"))
	state.ServerAuthAlgo = strValue(getString(obj, "serverauthalgo"))
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *servicesNTPSettingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan servicesNTPSettingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || r.client == nil {
		return
	}
	if _, err := r.client.Update(ctx, servicesNTPSettingsPath, applyNow(r.payload(plan))); err != nil {
		resp.Diagnostics.AddError("failed to update NTP settings", err.Error())
		return
	}
	plan.ID = types.StringValue("system")
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *servicesNTPSettingsResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// NTP settings cannot be deleted; removal from state is sufficient.
}

// ImportState adopts the settings already on the box. Singletons have a fixed
// identifier, so the import ID is always `system`.
func (r *servicesNTPSettingsResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
