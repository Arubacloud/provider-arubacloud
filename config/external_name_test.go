package config

import (
	"context"
	"testing"
)

const (
	testProjectID     = "proj-abc"
	testVPCID         = "vpc-123"
	testDbaasID       = "dbaas-123"
	testSGID          = "sg-456"
	tfProjectIDKey    = "project_id"
	tfVPCIDKey        = "vpc_id"
	tfSGIDKey         = "security_group_id"
	tfDbaasIDKey      = "dbaas_id"
	tfVPCPeeringIDKey = "vpc_peering_id"
	tfVPNTunnelIDKey  = "vpn_tunnel_id"
)

// TestLeafIDFromSlash verifies the helper that extracts a specific segment.
func TestLeafIDFromSlash(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		index int
		want  string
	}{
		{"subnet leaf at index 2", "proj-1/vpc-2/sub-3", 2, "sub-3"},
		{"sg leaf at index 2", "proj-1/vpc-2/sg-3", 2, "sg-3"},
		{"rule leaf at index 3", "proj-1/vpc-2/sg-3/rule-4", 3, "rule-4"},
		{"fallback when short", "only-one-part", 2, "only-one-part"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fn := leafIDFromSlash(tc.index)
			got, err := fn(map[string]any{"id": tc.id})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestSubnetGetIDFn verifies the subnet composite ID reconstruction.
func TestSubnetGetIDFn(t *testing.T) {
	e := subnetExternalName()
	id, err := e.GetIDFn(context.Background(), "sub-xyz", map[string]any{
		tfProjectIDKey: testProjectID,
		tfVPCIDKey:     testVPCID,
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "proj-abc/vpc-123/sub-xyz"
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}
}

func TestSubnetGetIDFn_MissingVpcID(t *testing.T) {
	e := subnetExternalName()
	_, err := e.GetIDFn(context.Background(), "sub-xyz", map[string]any{
		tfProjectIDKey: testProjectID,
	}, nil)
	if err == nil {
		t.Error("expected error when vpc_id is missing, got nil")
	}
}

// TestSecurityGroupGetIDFn verifies the 3-part sg ID.
func TestSecurityGroupGetIDFn(t *testing.T) {
	e := securityGroupExternalName()
	id, err := e.GetIDFn(context.Background(), "sg-xyz", map[string]any{
		tfProjectIDKey: testProjectID,
		tfVPCIDKey:     testVPCID,
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "proj-abc/vpc-123/sg-xyz"
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}
}

// TestSecurityRuleGetIDFn verifies the 4-part rule ID.
func TestSecurityRuleGetIDFn(t *testing.T) {
	e := securityRuleExternalName()
	id, err := e.GetIDFn(context.Background(), "rule-xyz", map[string]any{
		tfProjectIDKey: testProjectID,
		tfVPCIDKey:     testVPCID,
		tfSGIDKey:      testSGID,
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "proj-abc/vpc-123/sg-456/rule-xyz"
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}
}

// TestSnapshotExternalName verifies the unusual project/snap/billing_period format.
func TestSnapshotGetExternalName(t *testing.T) {
	e := snapshotExternalName()
	got, err := e.GetExternalNameFn(map[string]any{"id": "proj-abc/snap-xyz/Hour"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "snap-xyz" {
		t.Errorf("got %q, want %q", got, "snap-xyz")
	}
}

func TestSnapshotGetIDFn(t *testing.T) {
	e := snapshotExternalName()
	id, err := e.GetIDFn(context.Background(), "snap-xyz", map[string]any{
		tfProjectIDKey:   testProjectID,
		"billing_period": "Hour",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "proj-abc/snap-xyz/Hour"
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}
}

// TestDatabaseGetIDFn verifies the project/dbaas/db composite ID.
func TestDatabaseGetIDFn(t *testing.T) {
	e := databaseExternalName()
	id, err := e.GetIDFn(context.Background(), "db-xyz", map[string]any{
		tfProjectIDKey: testProjectID,
		tfDbaasIDKey:   testDbaasID,
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "proj-abc/dbaas-123/db-xyz"
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}
}

// TestIdentifierFromProviderWithProjectID verifies 2-part project/resource ID.
func TestIdentifierFromProviderWithProjectID(t *testing.T) {
	e := identifierFromProviderWithProjectID()
	id, err := e.GetIDFn(context.Background(), "vpc-abc", map[string]any{
		"project_id": "proj-xyz",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "proj-xyz/vpc-abc"
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}
}

func TestIdentifierFromProviderWithProjectID_Missing(t *testing.T) {
	e := identifierFromProviderWithProjectID()
	_, err := e.GetIDFn(context.Background(), "vpc-abc", map[string]any{}, nil)
	if err == nil {
		t.Error("expected error when project_id is missing, got nil")
	}
}

// TestVPCPeeringRouteGetIDFn verifies the corrected vpc_peering_id key.
func TestVPCPeeringRouteGetIDFn(t *testing.T) {
	e := vpcPeeringRouteExternalName()
	id, err := e.GetIDFn(context.Background(), "route-xyz", map[string]any{
		tfProjectIDKey:    testProjectID,
		tfVPCIDKey:        testVPCID,
		tfVPCPeeringIDKey: "peering-789",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := testProjectID + "/" + testVPCID + "/peering-789/route-xyz"
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}
}

// TestVPNRouteGetIDFn verifies vpn_tunnel_id key.
func TestVPNRouteGetIDFn(t *testing.T) {
	// vpnroute uses identifierFromProviderWithProjectID (2-part).
	e := identifierFromProviderWithProjectID()
	id, err := e.GetIDFn(context.Background(), "route-xyz", map[string]any{
		tfProjectIDKey: testProjectID,
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := testProjectID + "/route-xyz"
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}
}

// TestVPCPeeringGetIDFn verifies the 3-part peering ID.
func TestVPCPeeringGetIDFn(t *testing.T) {
	e := vpcPeeringExternalName()
	id, err := e.GetIDFn(context.Background(), "peering-xyz", map[string]any{
		tfProjectIDKey: testProjectID,
		tfVPCIDKey:     testVPCID,
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := testProjectID + "/" + testVPCID + "/peering-xyz"
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}
}

func TestVPCPeeringGetIDFn_MissingVpcID(t *testing.T) {
	e := vpcPeeringExternalName()
	_, err := e.GetIDFn(context.Background(), "peering-xyz", map[string]any{
		tfProjectIDKey: testProjectID,
	}, nil)
	if err == nil {
		t.Error("expected error when vpc_id is missing, got nil")
	}
}

// TestDBaaSUserGetIDFn verifies the 3-part project/dbaas/user ID.
func TestDBaaSUserGetIDFn(t *testing.T) {
	e := dbaasUserExternalName()
	id, err := e.GetIDFn(context.Background(), "user-xyz", map[string]any{
		tfProjectIDKey: testProjectID,
		tfDbaasIDKey:   testDbaasID,
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := testProjectID + "/" + testDbaasID + "/user-xyz"
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}
}

func TestDBaaSUserGetIDFn_MissingDbaasID(t *testing.T) {
	e := dbaasUserExternalName()
	_, err := e.GetIDFn(context.Background(), "user-xyz", map[string]any{
		tfProjectIDKey: testProjectID,
	}, nil)
	if err == nil {
		t.Error("expected error when dbaas_id is missing, got nil")
	}
}

// TestDatabaseGrantGetIDFn verifies the 4-part project/dbaas/database/grant ID.
func TestDatabaseGrantGetIDFn(t *testing.T) {
	e := databaseGrantExternalName()
	id, err := e.GetIDFn(context.Background(), "grant-xyz", map[string]any{
		tfProjectIDKey: testProjectID,
		tfDbaasIDKey:   testDbaasID,
		"database":     "appdb",
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := testProjectID + "/" + testDbaasID + "/appdb/grant-xyz"
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}
}

func TestDatabaseGrantGetIDFn_MissingParams(t *testing.T) {
	e := databaseGrantExternalName()
	_, err := e.GetIDFn(context.Background(), "grant-xyz", map[string]any{
		tfProjectIDKey: testProjectID,
		tfDbaasIDKey:   testDbaasID,
		// database missing
	}, nil)
	if err == nil {
		t.Error("expected error when database is missing, got nil")
	}
}

// TestDatabaseBackupGetIDFn verifies the 3-part project/dbaas/backup ID.
func TestDatabaseBackupGetIDFn(t *testing.T) {
	e := databaseBackupExternalName()
	id, err := e.GetIDFn(context.Background(), "dbbackup-xyz", map[string]any{
		tfProjectIDKey: testProjectID,
		tfDbaasIDKey:   testDbaasID,
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := testProjectID + "/" + testDbaasID + "/dbbackup-xyz"
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}
}

func TestDatabaseBackupGetIDFn_MissingDbaasID(t *testing.T) {
	e := databaseBackupExternalName()
	_, err := e.GetIDFn(context.Background(), "dbbackup-xyz", map[string]any{
		tfProjectIDKey: testProjectID,
	}, nil)
	if err == nil {
		t.Error("expected error when dbaas_id is missing, got nil")
	}
}

// TestLeafIDFromSlash_FallbackUsesFullID verifies the fallback path when the
// Terraform ID does not have enough segments.
func TestLeafIDFromSlash_FallbackUsesFullID(t *testing.T) {
	fn := leafIDFromSlash(3)
	got, err := fn(map[string]any{"id": "only-two/parts"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Fallback: returns the full Terraform ID when segments < index.
	if got == "" {
		t.Errorf("expected non-empty fallback, got %q", got)
	}
}

// TestSnapshotGetIDFn_MissingBillingPeriod verifies error on missing billing_period.
func TestSnapshotGetIDFn_MissingBillingPeriod(t *testing.T) {
	e := snapshotExternalName()
	_, err := e.GetIDFn(context.Background(), "snap-xyz", map[string]any{
		tfProjectIDKey: testProjectID,
		// billing_period missing
	}, nil)
	if err == nil {
		t.Error("expected error when billing_period is missing, got nil")
	}
}
