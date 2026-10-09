package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/superplanehq/superplane/pkg/fleets/adminclient"
	"github.com/superplanehq/superplane/pkg/fleets/artifact"
	fleetconfig "github.com/superplanehq/superplane/pkg/fleets/config"
	"github.com/superplanehq/superplane/pkg/fleets/provider"
	awsprovider "github.com/superplanehq/superplane/pkg/fleets/provider/aws"
	azureprovider "github.com/superplanehq/superplane/pkg/fleets/provider/azure"
	dockerprovider "github.com/superplanehq/superplane/pkg/fleets/provider/docker"
	gcpprovider "github.com/superplanehq/superplane/pkg/fleets/provider/gcp"
	"github.com/superplanehq/superplane/pkg/fleets/reconcile"
)

const (
	configPathEnvironment = "FLEET_MANAGER_CONFIG_FILE"
	defaultConfigPath     = "/etc/superplane/fleet-manager.yaml"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	configPath := os.Getenv(configPathEnvironment)
	if configPath == "" {
		configPath = defaultConfigPath
	}
	config, err := fleetconfig.Load(configPath)
	if err != nil {
		log.Error("load Fleet Manager config", slog.Any("error", err))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	httpClient := &http.Client{Timeout: config.RequestTimeout()}
	admin, err := adminclient.New(
		config.SuperPlaneURL,
		config.InstallationAdminToken,
		httpClient,
	)
	if err != nil {
		log.Error("create installation admin API client", slog.Any("error", err))
		os.Exit(1)
	}
	var releaseArtifactResolver reconcile.ArtifactResolver
	var awsSDKConfig aws.Config
	azureClients := map[string]azureprovider.ComputeAPI{}
	if usesReleaseArtifacts(config) {
		releaseArtifactResolver, err = artifact.NewResolver(
			config.RunnerReleaseBaseURL,
			httpClient,
		)
		if err != nil {
			log.Error("create runner artifact resolver", slog.Any("error", err))
			os.Exit(1)
		}
	}
	if usesAWS(config) {
		loadedAWSConfig, configErr := awsconfig.LoadDefaultConfig(ctx)
		if configErr != nil {
			log.Error("load AWS config", slog.Any("error", configErr))
			os.Exit(1)
		}
		awsSDKConfig = loadedAWSConfig
	}
	if usesAzure(config) {
		credential, credErr := azidentity.NewDefaultAzureCredential(nil)
		if credErr != nil {
			log.Error("create Azure credential", slog.Any("error", credErr))
			os.Exit(1)
		}
		for _, fleet := range config.Fleets {
			if fleet.Provider != fleetconfig.ProviderAzure {
				continue
			}
			if _, exists := azureClients[fleet.Azure.SubscriptionID]; exists {
				continue
			}
			client, clientErr := azureprovider.NewSDK(fleet.Azure.SubscriptionID, credential)
			if clientErr != nil {
				log.Error(
					"create Azure compute client",
					slog.String("subscription_id", fleet.Azure.SubscriptionID),
					slog.Any("error", clientErr),
				)
				os.Exit(1)
			}
			azureClients[fleet.Azure.SubscriptionID] = client
		}
	}
	var gcpClient gcpprovider.ComputeAPI
	if usesGCP(config) {
		gcpClient, err = gcpprovider.NewSDK(ctx)
		if err != nil {
			log.Error("create Compute Engine client", slog.Any("error", err))
			os.Exit(1)
		}
	}

	reconcilers := make([]*reconcile.Reconciler, 0, len(config.Fleets))
	for _, fleet := range config.Fleets {
		var ec2Client *ec2.Client
		if fleet.Provider == fleetconfig.ProviderAWS {
			ec2Client = newEC2Client(awsSDKConfig, fleet.AWS.Region)
		}
		resourceProvider, artifactResolver, architecture, err := buildProvider(
			config.ID,
			fleet,
			ec2Client,
			azureClients[fleet.Azure.SubscriptionID],
			gcpClient,
			releaseArtifactResolver,
			log,
		)
		if err != nil {
			log.Error(
				"create runner provider",
				slog.String("fleet_id", fleet.ID),
				slog.Any("error", err),
			)
			os.Exit(1)
		}
		fleetReconciler, err := reconcile.New(
			admin,
			artifactResolver,
			resourceProvider,
			reconcile.Config{
				FleetID:         fleet.ID,
				WarmCapacity:    fleet.WarmCapacity,
				MaxCapacity:     fleet.MaxCapacity,
				OperatingSystem: "linux",
				Architecture:    architecture,
				PollTimeout:     config.PollTimeout(),
			},
			log,
		)
		if err != nil {
			log.Error(
				"create fleet reconciler",
				slog.String("fleet_id", fleet.ID),
				slog.Any("error", err),
			)
			os.Exit(1)
		}
		reconcilers = append(reconcilers, fleetReconciler)
	}

	reconcile.Run(ctx, log, reconcile.RunConfig{
		Interval:           config.ReconciliationInterval(),
		ErrorRetryInterval: config.ErrorRetryInterval(),
	}, reconcilers)
	log.Info(
		"Fleet Manager started",
		slog.Int("fleet_count", len(reconcilers)),
	)
	<-ctx.Done()
	log.Info("Fleet Manager stopped")
}

func newEC2Client(config aws.Config, region string) *ec2.Client {
	config.Region = region
	return ec2.NewFromConfig(config)
}

func usesAWS(config *fleetconfig.Config) bool {
	return hasProvider(config, fleetconfig.ProviderAWS)
}

func usesAzure(config *fleetconfig.Config) bool {
	return hasProvider(config, fleetconfig.ProviderAzure)
}

func usesGCP(config *fleetconfig.Config) bool {
	return hasProvider(config, fleetconfig.ProviderGCP)
}

func usesReleaseArtifacts(config *fleetconfig.Config) bool {
	return usesAWS(config) || usesAzure(config) || usesGCP(config)
}

func hasProvider(config *fleetconfig.Config, providerName string) bool {
	for _, fleet := range config.Fleets {
		if fleet.Provider == providerName {
			return true
		}
	}
	return false
}

func buildProvider(
	fleetManagerID string,
	fleet fleetconfig.Fleet,
	ec2Client *ec2.Client,
	azureClient azureprovider.ComputeAPI,
	gcpClient gcpprovider.ComputeAPI,
	releaseArtifactResolver reconcile.ArtifactResolver,
	log *slog.Logger,
) (provider.Provider, reconcile.ArtifactResolver, string, error) {
	switch fleet.Provider {
	case fleetconfig.ProviderAWS:
		resourceProvider, err := awsprovider.New(ec2Client, awsprovider.Config{
			FleetManagerID:       fleetManagerID,
			AMI:                  fleet.AWS.AMI,
			InstanceType:         fleet.AWS.InstanceType,
			Architecture:         fleet.AWS.Architecture,
			SubnetIDs:            fleet.AWS.SubnetIDs,
			SecurityGroupIDs:     fleet.AWS.SecurityGroupIDs,
			IAMInstanceProfile:   fleet.AWS.IAMInstanceProfile,
			KeyName:              fleet.AWS.KeyName,
			VolumeSizeGB:         fleet.AWS.VolumeSizeGB,
			VolumeIOPS:           fleet.AWS.VolumeIOPS,
			VolumeThroughputMBps: fleet.AWS.VolumeThroughputMBps,
			ResourceTags:         fleet.AWS.ResourceTags,
			CloudWatchRegion:     fleet.AWS.Region,
			CloudWatchLogGroup:   fleet.AWS.CloudWatch.LogGroupName,
		}, log)
		return resourceProvider, releaseArtifactResolver, fleet.AWS.Architecture, err
	case fleetconfig.ProviderAzure:
		resourceProvider, err := azureprovider.New(azureClient, azureprovider.Config{
			FleetManagerID:         fleetManagerID,
			SubscriptionID:         fleet.Azure.SubscriptionID,
			ResourceGroup:          fleet.Azure.ResourceGroup,
			Location:               fleet.Azure.Location,
			ImageID:                fleet.Azure.ImageID,
			VMSize:                 fleet.Azure.VMSize,
			Architecture:           fleet.Azure.Architecture,
			SubnetID:               fleet.Azure.SubnetID,
			NetworkSecurityGroupID: fleet.Azure.NetworkSecurityGroupID,
			IdentityID:             fleet.Azure.IdentityID,
			Zones:                  fleet.Azure.Zones,
			DiskSizeGB:             fleet.Azure.DiskSizeGB,
			EphemeralOSDisk:        fleet.Azure.EphemeralOSDisk,
			ResourceTags:           fleet.Azure.ResourceTags,
		}, log)
		return resourceProvider, releaseArtifactResolver, fleet.Azure.Architecture, err
	case fleetconfig.ProviderGCP:
		resourceProvider, err := gcpprovider.New(gcpClient, gcpprovider.Config{
			FleetManagerID:      fleetManagerID,
			ProjectID:           fleet.GCP.ProjectID,
			Zones:               fleet.GCP.Zones,
			MachineType:         fleet.GCP.MachineType,
			Image:               fleet.GCP.Image,
			Architecture:        fleet.GCP.Architecture,
			Subnetwork:          fleet.GCP.Subnetwork,
			ServiceAccountEmail: fleet.GCP.ServiceAccountEmail,
			NetworkTags:         fleet.GCP.NetworkTags,
			DiskSizeGB:          fleet.GCP.DiskSizeGB,
			DiskType:            fleet.GCP.DiskType,
			Labels:              fleet.GCP.Labels,
		}, log)
		return resourceProvider, releaseArtifactResolver, fleet.GCP.Architecture, err
	case fleetconfig.ProviderDocker:
		resourceProvider, err := dockerprovider.New(dockerprovider.Config{
			Image:        fleet.Docker.Image,
			RunnerAPIURL: fleet.Docker.RunnerAPIURL,
			Network:      fleet.Docker.Network,
			Volumes:      fleet.Docker.Volumes,
			ExtraHosts:   fleet.Docker.ExtraHosts,
		}, log)
		return resourceProvider, emptyArtifactResolver{},
			fleet.Docker.Architecture, err
	default:
		return nil, nil, "", fmt.Errorf("unknown provider %q", fleet.Provider)
	}
}

type emptyArtifactResolver struct{}

func (emptyArtifactResolver) Resolve(
	context.Context,
	string,
	string,
	string,
) (artifact.Artifact, error) {
	return artifact.Artifact{}, nil
}
