package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/superplanehq/superplane/pkg/fleets/adminclient"
	"github.com/superplanehq/superplane/pkg/fleets/artifact"
	fleetconfig "github.com/superplanehq/superplane/pkg/fleets/config"
	"github.com/superplanehq/superplane/pkg/fleets/provider"
	awsprovider "github.com/superplanehq/superplane/pkg/fleets/provider/aws"
	dockerprovider "github.com/superplanehq/superplane/pkg/fleets/provider/docker"
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
	var awsArtifactResolver reconcile.ArtifactResolver
	var awsSDKConfig aws.Config
	if usesAWS(config) {
		awsArtifactResolver, err = artifact.NewResolver(
			config.RunnerReleaseBaseURL,
			httpClient,
		)
		if err != nil {
			log.Error("create runner artifact resolver", slog.Any("error", err))
			os.Exit(1)
		}
		loadedAWSConfig, configErr := awsconfig.LoadDefaultConfig(ctx)
		if configErr != nil {
			log.Error("load AWS config", slog.Any("error", configErr))
			os.Exit(1)
		}
		awsSDKConfig = loadedAWSConfig
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
			awsArtifactResolver,
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
				FleetManagerID:  config.ID,
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
	for _, fleet := range config.Fleets {
		if fleet.Provider == fleetconfig.ProviderAWS {
			return true
		}
	}
	return false
}

func buildProvider(
	fleetManagerID string,
	fleet fleetconfig.Fleet,
	ec2Client *ec2.Client,
	awsArtifactResolver reconcile.ArtifactResolver,
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
		return resourceProvider, awsArtifactResolver, fleet.AWS.Architecture, err
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
