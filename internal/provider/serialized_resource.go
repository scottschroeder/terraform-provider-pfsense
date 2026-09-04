package provider

import (
	"context"
	"fmt"

	"github.com/elacy/terraform-provider-pfsense/v2/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// configurableImportableResource is the common interface implemented by every
// managed resource in this provider.
type configurableImportableResource interface {
	resource.Resource
	resource.ResourceWithConfigure
	resource.ResourceWithImportState
}

// serializedResource holds the pfSense mutation gate across the complete
// Terraform Create, Update, or Delete operation. This includes any natural-key
// and parent lookups performed before the final write, which is required
// because many pfSense REST API IDs are mutable array indexes.
type serializedResource struct {
	inner  configurableImportableResource
	client *client.Client
}

var _ resource.ResourceWithConfigure = (*serializedResource)(nil)
var _ resource.ResourceWithImportState = (*serializedResource)(nil)

func newSerializedResource(inner resource.Resource) resource.Resource {
	standard, ok := inner.(configurableImportableResource)
	if !ok {
		panic(fmt.Sprintf("resource %T must implement configure and import", inner))
	}
	wrapped := &serializedResource{inner: standard}
	if upgradable, ok := inner.(resource.ResourceWithUpgradeState); ok {
		return &serializedUpgradableResource{
			serializedResource: wrapped,
			upgradable:         upgradable,
		}
	}
	return wrapped
}

func serializeResourceConstructors(constructors []func() resource.Resource) []func() resource.Resource {
	wrapped := make([]func() resource.Resource, len(constructors))
	for i, constructor := range constructors {
		constructor := constructor
		wrapped[i] = func() resource.Resource {
			return newSerializedResource(constructor())
		}
	}
	return wrapped
}

func (r *serializedResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	r.inner.Metadata(ctx, req, resp)
}

func (r *serializedResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	r.inner.Schema(ctx, req, resp)
}

func (r *serializedResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.inner.Configure(ctx, req, resp)
	if configuredClient, ok := req.ProviderData.(*client.Client); ok {
		r.client = configuredClient
	}
}

func (r *serializedResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	mutationCtx, release, ok := r.beginMutation(ctx, &resp.Diagnostics)
	if !ok {
		return
	}
	defer release()
	r.inner.Create(mutationCtx, req, resp)
}

func (r *serializedResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	r.inner.Read(ctx, req, resp)
}

func (r *serializedResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	mutationCtx, release, ok := r.beginMutation(ctx, &resp.Diagnostics)
	if !ok {
		return
	}
	defer release()
	r.inner.Update(mutationCtx, req, resp)
}

func (r *serializedResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	mutationCtx, release, ok := r.beginMutation(ctx, &resp.Diagnostics)
	if !ok {
		return
	}
	defer release()
	r.inner.Delete(mutationCtx, req, resp)
}

func (r *serializedResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	r.inner.ImportState(ctx, req, resp)
}

func (r *serializedResource) beginMutation(ctx context.Context, diagnostics interface {
	AddError(string, string)
}) (context.Context, func(), bool) {
	if r.client == nil {
		// Let the wrapped resource report its existing configuration diagnostic.
		return ctx, func() {}, true
	}
	mutationCtx, release, err := r.client.BeginMutation(ctx)
	if err != nil {
		diagnostics.AddError("failed to begin pfSense mutation", err.Error())
		return ctx, nil, false
	}
	return mutationCtx, release, true
}

// serializedUpgradableResource preserves the state-upgrade capability of the
// seven resources that implement it while sharing the serialized CRUD methods.
type serializedUpgradableResource struct {
	*serializedResource
	upgradable resource.ResourceWithUpgradeState
}

var _ resource.ResourceWithUpgradeState = (*serializedUpgradableResource)(nil)

func (r *serializedUpgradableResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return r.upgradable.UpgradeState(ctx)
}
