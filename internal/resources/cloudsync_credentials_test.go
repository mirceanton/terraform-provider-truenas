package resources

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestNewCloudSyncCredentialsResource(t *testing.T) {
	r := NewCloudSyncCredentialsResource()
	if r == nil {
		t.Fatal("expected non-nil resource")
	}

	var _ resource.Resource = r
	var _ resource.ResourceWithConfigure = r.(*CloudSyncCredentialsResource)
	var _ resource.ResourceWithImportState = r.(*CloudSyncCredentialsResource)
}
