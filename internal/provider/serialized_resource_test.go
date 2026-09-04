package provider

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/elacy/terraform-provider-pfsense/v2/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

type mutationProbe struct {
	active  atomic.Int64
	overlap atomic.Bool
}

func (p *mutationProbe) run() {
	if p.active.Add(1) > 1 {
		p.overlap.Store(true)
	}
	time.Sleep(10 * time.Millisecond)
	p.active.Add(-1)
}

type mutationProbeResource struct {
	probe *mutationProbe
}

var _ configurableImportableResource = (*mutationProbeResource)(nil)

func (r *mutationProbeResource) Metadata(context.Context, resource.MetadataRequest, *resource.MetadataResponse) {
}
func (r *mutationProbeResource) Schema(context.Context, resource.SchemaRequest, *resource.SchemaResponse) {
}
func (r *mutationProbeResource) Configure(context.Context, resource.ConfigureRequest, *resource.ConfigureResponse) {
}
func (r *mutationProbeResource) Create(context.Context, resource.CreateRequest, *resource.CreateResponse) {
	r.probe.run()
}
func (r *mutationProbeResource) Read(context.Context, resource.ReadRequest, *resource.ReadResponse) {
}
func (r *mutationProbeResource) Update(context.Context, resource.UpdateRequest, *resource.UpdateResponse) {
	r.probe.run()
}
func (r *mutationProbeResource) Delete(context.Context, resource.DeleteRequest, *resource.DeleteResponse) {
	r.probe.run()
}
func (r *mutationProbeResource) ImportState(context.Context, resource.ImportStateRequest, *resource.ImportStateResponse) {
}

func TestSerializedResourceCoversEntireMutations(t *testing.T) {
	configuredClient, err := client.New(client.Config{URL: "http://127.0.0.1", APIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	probe := &mutationProbe{}
	wrapped := make([]*serializedResource, 3)
	for i := range wrapped {
		wrapped[i] = newSerializedResource(&mutationProbeResource{probe: probe}).(*serializedResource)
		wrapped[i].client = configuredClient
	}

	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		<-start
		wrapped[0].Create(context.Background(), resource.CreateRequest{}, &resource.CreateResponse{})
	}()
	go func() {
		defer wg.Done()
		<-start
		wrapped[1].Update(context.Background(), resource.UpdateRequest{}, &resource.UpdateResponse{})
	}()
	go func() {
		defer wg.Done()
		<-start
		wrapped[2].Delete(context.Background(), resource.DeleteRequest{}, &resource.DeleteResponse{})
	}()
	close(start)
	wg.Wait()

	if probe.overlap.Load() {
		t.Fatal("complete resource mutation operations overlapped")
	}
}

func TestSerializedResourcePreservesProviderInterfaces(t *testing.T) {
	constructors := (&pfsenseProvider{}).Resources(context.Background())
	if len(constructors) != 63 {
		t.Fatalf("resource constructor count = %d, want 63", len(constructors))
	}

	upgradable := 0
	for _, constructor := range constructors {
		wrapped := constructor()
		if _, ok := wrapped.(resource.ResourceWithConfigure); !ok {
			t.Fatalf("wrapped resource %T lost ResourceWithConfigure", wrapped)
		}
		if _, ok := wrapped.(resource.ResourceWithImportState); !ok {
			t.Fatalf("wrapped resource %T lost ResourceWithImportState", wrapped)
		}
		if _, ok := wrapped.(resource.ResourceWithUpgradeState); ok {
			upgradable++
		}
	}
	if upgradable != 7 {
		t.Fatalf("upgradable wrapped resource count = %d, want 7", upgradable)
	}
}
