// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/upjet/v2/pkg/controller"

	backup "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/backup"
	blockstorage "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/blockstorage"
	cloudserver "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/cloudserver"
	containerregistry "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/containerregistry"
	database "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/database"
	databasebackup "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/databasebackup"
	databasegrant "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/databasegrant"
	dbaas "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/dbaas"
	dbaasuser "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/dbaasuser"
	elasticip "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/elasticip"
	kaas "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/kaas"
	keypair "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/keypair"
	kms "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/kms"
	project "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/project"
	restore "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/restore"
	schedulejob "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/schedulejob"
	securitygroup "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/securitygroup"
	securityrule "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/securityrule"
	snapshot "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/snapshot"
	subnet "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/subnet"
	vpc "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/vpc"
	vpcpeering "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/vpcpeering"
	vpcpeeringroute "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/vpcpeeringroute"
	vpnroute "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/vpnroute"
	vpntunnel "github.com/arubacloud/provider-arubacloud/internal/controller/arubacloud/vpntunnel"
	providerconfig "github.com/arubacloud/provider-arubacloud/internal/controller/providerconfig"
)

// Setup creates all controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		backup.Setup,
		blockstorage.Setup,
		cloudserver.Setup,
		containerregistry.Setup,
		database.Setup,
		databasebackup.Setup,
		databasegrant.Setup,
		dbaas.Setup,
		dbaasuser.Setup,
		elasticip.Setup,
		kaas.Setup,
		keypair.Setup,
		kms.Setup,
		project.Setup,
		restore.Setup,
		schedulejob.Setup,
		securitygroup.Setup,
		securityrule.Setup,
		snapshot.Setup,
		subnet.Setup,
		vpc.Setup,
		vpcpeering.Setup,
		vpcpeeringroute.Setup,
		vpnroute.Setup,
		vpntunnel.Setup,
		providerconfig.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// SetupGated creates all controllers with the supplied logger and adds them to
// the supplied manager gated.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		backup.SetupGated,
		blockstorage.SetupGated,
		cloudserver.SetupGated,
		containerregistry.SetupGated,
		database.SetupGated,
		databasebackup.SetupGated,
		databasegrant.SetupGated,
		dbaas.SetupGated,
		dbaasuser.SetupGated,
		elasticip.SetupGated,
		kaas.SetupGated,
		keypair.SetupGated,
		kms.SetupGated,
		project.SetupGated,
		restore.SetupGated,
		schedulejob.SetupGated,
		securitygroup.SetupGated,
		securityrule.SetupGated,
		snapshot.SetupGated,
		subnet.SetupGated,
		vpc.SetupGated,
		vpcpeering.SetupGated,
		vpcpeeringroute.SetupGated,
		vpnroute.SetupGated,
		vpntunnel.SetupGated,
		providerconfig.SetupGated,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// SetupWebhookWithManager registers conversion webhooks for all resource kinds in the group.
func SetupWebhookWithManager(mgr ctrl.Manager) error {
	for _, setup := range []func(ctrl.Manager) error{
		backup.SetupWebhookWithManager,
		blockstorage.SetupWebhookWithManager,
		cloudserver.SetupWebhookWithManager,
		containerregistry.SetupWebhookWithManager,
		database.SetupWebhookWithManager,
		databasebackup.SetupWebhookWithManager,
		databasegrant.SetupWebhookWithManager,
		dbaas.SetupWebhookWithManager,
		dbaasuser.SetupWebhookWithManager,
		elasticip.SetupWebhookWithManager,
		kaas.SetupWebhookWithManager,
		keypair.SetupWebhookWithManager,
		kms.SetupWebhookWithManager,
		project.SetupWebhookWithManager,
		restore.SetupWebhookWithManager,
		schedulejob.SetupWebhookWithManager,
		securitygroup.SetupWebhookWithManager,
		securityrule.SetupWebhookWithManager,
		snapshot.SetupWebhookWithManager,
		subnet.SetupWebhookWithManager,
		vpc.SetupWebhookWithManager,
		vpcpeering.SetupWebhookWithManager,
		vpcpeeringroute.SetupWebhookWithManager,
		vpnroute.SetupWebhookWithManager,
		vpntunnel.SetupWebhookWithManager,
		providerconfig.SetupWebhookWithManager,
	} {
		if err := setup(mgr); err != nil {
			return err
		}
	}
	return nil
}
