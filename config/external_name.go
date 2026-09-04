package config

import (
	"context"
	"fmt"
	"strings"

	"github.com/crossplane/upjet/v2/pkg/config"
)

// ExternalNameConfigs contains all external name configurations for this
// provider. All ArubaCloud resources use provider-generated IDs, so the
// default is IdentifierFromProvider. Resources with composite Terraform import
// IDs (e.g. project_id/vpc_id/subnet_id) need custom GetIDFn to reconstruct
// the full import path from the leaf external name and resolved references.
var ExternalNameConfigs = map[string]config.ExternalName{
	// ─── Simple 2-part import ID (project_id/resource_id) ──────────────────
	"arubacloud_project":           config.IdentifierFromProvider,
	"arubacloud_vpc":               identifierFromProviderWithProjectID(),
	"arubacloud_keypair":           identifierFromProviderWithProjectID(),
	"arubacloud_elasticip":         identifierFromProviderWithProjectID(),
	"arubacloud_blockstorage":      identifierFromProviderWithProjectID(),
	"arubacloud_backup":            identifierFromProviderWithProjectID(),
	"arubacloud_restore":           identifierFromProviderWithProjectID(),
	"arubacloud_kaas":              identifierFromProviderWithProjectID(),
	"arubacloud_containerregistry": identifierFromProviderWithProjectID(),
	"arubacloud_dbaas":             identifierFromProviderWithProjectID(),
	"arubacloud_vpntunnel":         identifierFromProviderWithProjectID(),
	"arubacloud_vpnroute":          identifierFromProviderWithProjectID(),
	"arubacloud_schedulejob":       identifierFromProviderWithProjectID(),
	"arubacloud_kms":               identifierFromProviderWithProjectID(),
	"arubacloud_cloudserver":       identifierFromProviderWithProjectID(),

	// ─── 3-part: project_id/vpc_id/resource_id ─────────────────────────────
	"arubacloud_subnet":          subnetExternalName(),
	"arubacloud_securitygroup":   securityGroupExternalName(),
	"arubacloud_vpcpeering":      vpcPeeringExternalName(),
	"arubacloud_vpcpeeringroute": vpcPeeringRouteExternalName(),

	// ─── 4-part: project_id/vpc_id/sg_id/rule_id ───────────────────────────
	"arubacloud_securityrule": securityRuleExternalName(),

	// ─── 3-part: project_id/snap_id/billing_period ─────────────────────────
	"arubacloud_snapshot": snapshotExternalName(),

	// ─── 3-part: project_id/dbaas_id/resource_id ───────────────────────────
	"arubacloud_database":       databaseExternalName(),
	"arubacloud_dbaasuser":      dbaasUserExternalName(),
	"arubacloud_databasegrant":  databaseGrantExternalName(),
	"arubacloud_databasebackup": databaseBackupExternalName(),
}

// identifierFromProviderWithProjectID returns an external name config where the
// leaf resource ID is the external name and the full Terraform import ID is
// project_id/resource_id.
func identifierFromProviderWithProjectID() config.ExternalName {
	e := config.IdentifierFromProvider
	e.GetIDFn = func(_ context.Context, externalName string, parameters map[string]any, _ map[string]any) (string, error) {
		projectID, ok := parameters["project_id"].(string)
		if !ok || projectID == "" {
			return "", fmt.Errorf("project_id is required")
		}
		return projectID + "/" + externalName, nil
	}
	e.DisableNameInitializer = true
	return e
}

// subnetExternalName handles arubacloud_subnet with import ID:
// project_id/vpc_id/subnet_id
func subnetExternalName() config.ExternalName {
	e := config.IdentifierFromProvider
	e.GetExternalNameFn = leafIDFromSlash(2)
	e.GetIDFn = func(_ context.Context, externalName string, parameters map[string]any, _ map[string]any) (string, error) {
		projectID, _ := parameters["project_id"].(string)
		vpcID, _ := parameters["vpc_id"].(string)
		if projectID == "" || vpcID == "" {
			return "", fmt.Errorf("project_id and vpc_id are required for subnet import ID")
		}
		return projectID + "/" + vpcID + "/" + externalName, nil
	}
	e.DisableNameInitializer = true
	return e
}

// securityGroupExternalName handles arubacloud_securitygroup with import ID:
// project_id/vpc_id/sg_id
func securityGroupExternalName() config.ExternalName {
	e := config.IdentifierFromProvider
	e.GetExternalNameFn = leafIDFromSlash(2)
	e.GetIDFn = func(_ context.Context, externalName string, parameters map[string]any, _ map[string]any) (string, error) {
		projectID, _ := parameters["project_id"].(string)
		vpcID, _ := parameters["vpc_id"].(string)
		if projectID == "" || vpcID == "" {
			return "", fmt.Errorf("project_id and vpc_id are required for security group import ID")
		}
		return projectID + "/" + vpcID + "/" + externalName, nil
	}
	e.DisableNameInitializer = true
	return e
}

// vpcPeeringExternalName handles arubacloud_vpcpeering with import ID:
// project_id/vpc_id/peering_id
func vpcPeeringExternalName() config.ExternalName {
	e := config.IdentifierFromProvider
	e.GetExternalNameFn = leafIDFromSlash(2)
	e.GetIDFn = func(_ context.Context, externalName string, parameters map[string]any, _ map[string]any) (string, error) {
		projectID, _ := parameters["project_id"].(string)
		vpcID, _ := parameters["vpc_id"].(string)
		if projectID == "" || vpcID == "" {
			return "", fmt.Errorf("project_id and vpc_id are required for VPC peering import ID")
		}
		return projectID + "/" + vpcID + "/" + externalName, nil
	}
	e.DisableNameInitializer = true
	return e
}

// vpcPeeringRouteExternalName handles arubacloud_vpcpeeringroute.
// Import format: project_id/vpc_id/vpc_peering_id/route_id (verified from generated types).
func vpcPeeringRouteExternalName() config.ExternalName {
	e := config.IdentifierFromProvider
	e.GetExternalNameFn = leafIDFromSlash(3)
	e.GetIDFn = func(_ context.Context, externalName string, parameters map[string]any, _ map[string]any) (string, error) {
		projectID, _ := parameters["project_id"].(string)
		vpcID, _ := parameters["vpc_id"].(string)
		peeringID, _ := parameters["vpc_peering_id"].(string)
		if projectID == "" || vpcID == "" || peeringID == "" {
			return "", fmt.Errorf("project_id, vpc_id, and vpc_peering_id are required for VPC peering route")
		}
		return projectID + "/" + vpcID + "/" + peeringID + "/" + externalName, nil
	}
	e.DisableNameInitializer = true
	return e
}

// securityRuleExternalName handles arubacloud_securityrule with import ID:
// project_id/vpc_id/sg_id/rule_id
func securityRuleExternalName() config.ExternalName {
	e := config.IdentifierFromProvider
	e.GetExternalNameFn = leafIDFromSlash(3)
	e.GetIDFn = func(_ context.Context, externalName string, parameters map[string]any, _ map[string]any) (string, error) {
		projectID, _ := parameters["project_id"].(string)
		vpcID, _ := parameters["vpc_id"].(string)
		sgID, _ := parameters["security_group_id"].(string)
		if projectID == "" || vpcID == "" || sgID == "" {
			return "", fmt.Errorf("project_id, vpc_id, and security_group_id are required for security rule import ID")
		}
		return projectID + "/" + vpcID + "/" + sgID + "/" + externalName, nil
	}
	e.DisableNameInitializer = true
	return e
}

// snapshotExternalName handles arubacloud_snapshot with import ID:
// project_id/snapshot_id/billing_period
func snapshotExternalName() config.ExternalName {
	e := config.IdentifierFromProvider
	e.GetExternalNameFn = func(tfstate map[string]any) (string, error) {
		// Terraform ID is project_id/snapshot_id/billing_period — extract snapshot_id
		id, _ := tfstate["id"].(string)
		parts := strings.Split(id, "/")
		if len(parts) == 3 {
			return parts[1], nil
		}
		en, _ := config.IDAsExternalName(tfstate)
		return en, nil
	}
	e.GetIDFn = func(_ context.Context, externalName string, parameters map[string]any, _ map[string]any) (string, error) {
		projectID, _ := parameters["project_id"].(string)
		billingPeriod, _ := parameters["billing_period"].(string)
		if projectID == "" || billingPeriod == "" {
			return "", fmt.Errorf("project_id and billing_period are required for snapshot import ID")
		}
		return projectID + "/" + externalName + "/" + billingPeriod, nil
	}
	e.DisableNameInitializer = true
	return e
}

// databaseExternalName handles arubacloud_database with import ID:
// project_id/dbaas_id/database_id
func databaseExternalName() config.ExternalName {
	e := config.IdentifierFromProvider
	e.GetExternalNameFn = leafIDFromSlash(2)
	e.GetIDFn = func(_ context.Context, externalName string, parameters map[string]any, _ map[string]any) (string, error) {
		projectID, _ := parameters["project_id"].(string)
		dbaasID, _ := parameters["dbaas_id"].(string)
		if projectID == "" || dbaasID == "" {
			return "", fmt.Errorf("project_id and dbaas_id are required for database import ID")
		}
		return projectID + "/" + dbaasID + "/" + externalName, nil
	}
	e.DisableNameInitializer = true
	return e
}

// dbaasUserExternalName handles arubacloud_dbaasuser with import ID:
// project_id/dbaas_id/user_id
func dbaasUserExternalName() config.ExternalName {
	e := config.IdentifierFromProvider
	e.GetExternalNameFn = leafIDFromSlash(2)
	e.GetIDFn = func(_ context.Context, externalName string, parameters map[string]any, _ map[string]any) (string, error) {
		projectID, _ := parameters["project_id"].(string)
		dbaasID, _ := parameters["dbaas_id"].(string)
		if projectID == "" || dbaasID == "" {
			return "", fmt.Errorf("project_id and dbaas_id are required for DBaaS user import ID")
		}
		return projectID + "/" + dbaasID + "/" + externalName, nil
	}
	e.DisableNameInitializer = true
	return e
}

// databaseGrantExternalName handles arubacloud_databasegrant.
// Import format: project_id/dbaas_id/database/user_id
func databaseGrantExternalName() config.ExternalName {
	e := config.IdentifierFromProvider
	e.GetExternalNameFn = leafIDFromSlash(3)
	e.GetIDFn = func(_ context.Context, externalName string, parameters map[string]any, _ map[string]any) (string, error) {
		projectID, _ := parameters["project_id"].(string)
		dbaasID, _ := parameters["dbaas_id"].(string)
		database, _ := parameters["database"].(string)
		if projectID == "" || dbaasID == "" || database == "" {
			return "", fmt.Errorf("project_id, dbaas_id, and database are required for database grant import ID")
		}
		return projectID + "/" + dbaasID + "/" + database + "/" + externalName, nil
	}
	e.DisableNameInitializer = true
	return e
}

// databaseBackupExternalName handles arubacloud_databasebackup.
// Import format REQUIRES VERIFICATION.
func databaseBackupExternalName() config.ExternalName {
	e := config.IdentifierFromProvider
	e.GetExternalNameFn = leafIDFromSlash(2)
	e.GetIDFn = func(_ context.Context, externalName string, parameters map[string]any, _ map[string]any) (string, error) {
		projectID, _ := parameters["project_id"].(string)
		dbaasID, _ := parameters["dbaas_id"].(string)
		if projectID == "" || dbaasID == "" {
			return "", fmt.Errorf("project_id and dbaas_id are required (REQUIRES VERIFICATION)")
		}
		return projectID + "/" + dbaasID + "/" + externalName, nil
	}
	e.DisableNameInitializer = true
	return e
}

// leafIDFromSlash returns a GetExternalNameFn that extracts the segment at
// the given index from a slash-delimited Terraform ID.
func leafIDFromSlash(index int) func(map[string]any) (string, error) {
	return func(tfstate map[string]any) (string, error) {
		id, _ := tfstate["id"].(string)
		parts := strings.Split(id, "/")
		if len(parts) > index {
			return parts[index], nil
		}
		en, _ := config.IDAsExternalName(tfstate)
		return en, nil
	}
}

// ExternalNameConfigurations applies all external name configs listed in the
// table ExternalNameConfigs and sets the version of those resources to v1beta1
// assuming they will be tested.
func ExternalNameConfigurations() config.ResourceOption {
	return func(r *config.Resource) {
		if e, ok := ExternalNameConfigs[r.Name]; ok {
			r.ExternalName = e
		}
	}
}

// ExternalNameConfigured returns the list of all resources whose external name
// is configured manually.
func ExternalNameConfigured() []string {
	l := make([]string, len(ExternalNameConfigs))
	i := 0
	for name := range ExternalNameConfigs {
		// $ is added to match the exact string since the format is regex.
		l[i] = name + "$"
		i++
	}
	return l
}
