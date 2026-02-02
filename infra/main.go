package main

import (
	"fmt"

	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/artifactregistry"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/compute"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/firestore"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v7/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		// Configuration
		cfg := config.New(ctx, "namazu-infra")
		gcpCfg := config.New(ctx, "gcp")

		env := cfg.Require("environment")
		machineType := cfg.Get("machineType")
		if machineType == "" {
			machineType = "e2-micro"
		}
		domain := cfg.Get("domain")
		authTenantId := cfg.Get("authTenantId")

		project := gcpCfg.Require("project")
		region := gcpCfg.Get("region")
		if region == "" {
			region = "us-west1"
		}
		zone := gcpCfg.Get("zone")
		if zone == "" {
			zone = "us-west1-b"
		}

		// Resource naming
		namePrefix := fmt.Sprintf("namazu-%s", env)

		// Source endpoint: sandbox for non-prod, production for prod
		sourceEndpoint := "wss://api-realtime-sandbox.p2pquake.net/v2/ws"
		if env == "prod" {
			sourceEndpoint = "wss://api.p2pquake.net/v2/ws"
		}

		// =================================================================
		// Enable Required GCP APIs
		// =================================================================
		apis := map[string]string{
			"compute":          "compute.googleapis.com",
			"artifactregistry": "artifactregistry.googleapis.com",
			"firestore":        "firestore.googleapis.com",
			"iam":              "iam.googleapis.com",
		}

		enabledAPIs := make([]*projects.Service, 0, len(apis))
		for name, api := range apis {
			svc, err := projects.NewService(ctx, fmt.Sprintf("%s-enable-%s-api", namePrefix, name), &projects.ServiceArgs{
				Service:                  pulumi.String(api),
				DisableDependentServices: pulumi.Bool(false),
				DisableOnDestroy:         pulumi.Bool(false),
			})
			if err != nil {
				return err
			}
			enabledAPIs = append(enabledAPIs, svc)
		}

		// Create dependency array for resources that need APIs enabled first
		apiDeps := make([]pulumi.Resource, len(enabledAPIs))
		for i, api := range enabledAPIs {
			apiDeps[i] = api
		}

		// =================================================================
		// Artifact Registry - Docker image repository
		// =================================================================
		registry, err := artifactregistry.NewRepository(ctx, fmt.Sprintf("%s-registry", namePrefix), &artifactregistry.RepositoryArgs{
			RepositoryId: pulumi.String("namazu"),
			Location:     pulumi.String(region),
			Format:       pulumi.String("DOCKER"),
			Description:  pulumi.String("Docker images for namazu"),
		}, pulumi.DependsOn(apiDeps))
		if err != nil {
			return err
		}

		// =================================================================
		// Firestore Database
		// =================================================================
		// Database name: "namazu-stg" or "namazu-prod" (minimum 4 characters required)
		dbName := fmt.Sprintf("namazu-%s", env)
		firestoreDB, err := firestore.NewDatabase(ctx, fmt.Sprintf("%s-firestore", namePrefix), &firestore.DatabaseArgs{
			Name:                     pulumi.String(dbName),
			LocationId:               pulumi.String(region),
			Type:                     pulumi.String("FIRESTORE_NATIVE"),
			ConcurrencyMode:          pulumi.String("OPTIMISTIC"),
			AppEngineIntegrationMode: pulumi.String("DISABLED"),
		}, pulumi.DependsOn(apiDeps))
		if err != nil {
			return err
		}

		// =================================================================
		// Service Account for the application
		// =================================================================
		saName := fmt.Sprintf("namazu-%s-instance", env)
		serviceAccount, err := serviceaccount.NewAccount(ctx, fmt.Sprintf("%s-sa", namePrefix), &serviceaccount.AccountArgs{
			AccountId:   pulumi.String(saName),
			DisplayName: pulumi.String(fmt.Sprintf("Namazu %s Instance Service Account", env)),
			Description: pulumi.String("Service account for namazu instance"),
		}, pulumi.DependsOn(apiDeps))
		if err != nil {
			return err
		}

		// IAM bindings for the service account
		iamRoles := []struct {
			name string
			role string
		}{
			{"artifact-registry-reader", "roles/artifactregistry.reader"},
			{"firestore-user", "roles/datastore.user"},
			{"logging-writer", "roles/logging.logWriter"},
		}

		iamBindings := make([]pulumi.Resource, 0, len(iamRoles))
		for _, r := range iamRoles {
			binding, err := projects.NewIAMMember(ctx, fmt.Sprintf("%s-sa-%s", namePrefix, r.name), &projects.IAMMemberArgs{
				Project: pulumi.String(project),
				Role:    pulumi.String(r.role),
				Member:  pulumi.Sprintf("serviceAccount:%s", serviceAccount.Email),
			})
			if err != nil {
				return err
			}
			iamBindings = append(iamBindings, binding)
		}

		// =================================================================
		// VPC Network
		// =================================================================
		network, err := compute.NewNetwork(ctx, fmt.Sprintf("%s-network", namePrefix), &compute.NetworkArgs{
			AutoCreateSubnetworks: pulumi.Bool(false),
			Description:           pulumi.String("VPC network for namazu"),
		}, pulumi.DependsOn(apiDeps))
		if err != nil {
			return err
		}

		// Subnetwork
		subnet, err := compute.NewSubnetwork(ctx, fmt.Sprintf("%s-subnet", namePrefix), &compute.SubnetworkArgs{
			IpCidrRange: pulumi.String("10.0.0.0/24"),
			Region:      pulumi.String(region),
			Network:     network.ID(),
			Description: pulumi.String("Subnet for namazu instances"),
		})
		if err != nil {
			return err
		}

		// =================================================================
		// Firewall Rules
		// =================================================================
		// Allow HTTP/HTTPS from anywhere
		_, err = compute.NewFirewall(ctx, fmt.Sprintf("%s-allow-http", namePrefix), &compute.FirewallArgs{
			Network: network.Name,
			Allows: compute.FirewallAllowArray{
				&compute.FirewallAllowArgs{
					Protocol: pulumi.String("tcp"),
					Ports:    pulumi.StringArray{pulumi.String("80"), pulumi.String("443")},
				},
			},
			SourceRanges: pulumi.StringArray{pulumi.String("0.0.0.0/0")},
			TargetTags:   pulumi.StringArray{pulumi.String("namazu-server")},
			Description:  pulumi.String("Allow HTTP/HTTPS traffic to namazu"),
		})
		if err != nil {
			return err
		}

		// Allow SSH from IAP (Identity-Aware Proxy) for secure management
		_, err = compute.NewFirewall(ctx, fmt.Sprintf("%s-allow-iap-ssh", namePrefix), &compute.FirewallArgs{
			Network: network.Name,
			Allows: compute.FirewallAllowArray{
				&compute.FirewallAllowArgs{
					Protocol: pulumi.String("tcp"),
					Ports:    pulumi.StringArray{pulumi.String("22")},
				},
			},
			// IAP's IP range
			SourceRanges: pulumi.StringArray{pulumi.String("35.235.240.0/20")},
			TargetTags:   pulumi.StringArray{pulumi.String("namazu-server")},
			Description:  pulumi.String("Allow SSH via IAP to namazu"),
		})
		if err != nil {
			return err
		}

		// Allow internal health checks from GCP Load Balancer
		_, err = compute.NewFirewall(ctx, fmt.Sprintf("%s-allow-health-check", namePrefix), &compute.FirewallArgs{
			Network: network.Name,
			Allows: compute.FirewallAllowArray{
				&compute.FirewallAllowArgs{
					Protocol: pulumi.String("tcp"),
					Ports:    pulumi.StringArray{pulumi.String("8080")},
				},
			},
			// GCP health check ranges
			SourceRanges: pulumi.StringArray{
				pulumi.String("130.211.0.0/22"),
				pulumi.String("35.191.0.0/16"),
			},
			TargetTags:  pulumi.StringArray{pulumi.String("namazu-server")},
			Description: pulumi.String("Allow health checks from GCP"),
		})
		if err != nil {
			return err
		}

		// =================================================================
		// Static External IP
		// =================================================================
		staticIP, err := compute.NewAddress(ctx, fmt.Sprintf("%s-ip", namePrefix), &compute.AddressArgs{
			Region:      pulumi.String(region),
			AddressType: pulumi.String("EXTERNAL"),
			Description: pulumi.String("Static IP for namazu server"),
		})
		if err != nil {
			return err
		}

		// =================================================================
		// Cloud Router & NAT (for outbound connectivity)
		// =================================================================
		router, err := compute.NewRouter(ctx, fmt.Sprintf("%s-router", namePrefix), &compute.RouterArgs{
			Network: network.ID(),
			Region:  pulumi.String(region),
		})
		if err != nil {
			return err
		}

		_, err = compute.NewRouterNat(ctx, fmt.Sprintf("%s-nat", namePrefix), &compute.RouterNatArgs{
			Router:                        router.Name,
			Region:                        pulumi.String(region),
			NatIpAllocateOption:           pulumi.String("AUTO_ONLY"),
			SourceSubnetworkIpRangesToNat: pulumi.String("ALL_SUBNETWORKS_ALL_IP_RANGES"),
		})
		if err != nil {
			return err
		}

		// =================================================================
		// Compute Engine Instance
		// =================================================================
		// Startup script for Container-Optimized OS (Docker is pre-installed)
		// Configuration values are read from instance metadata
		startupScript := pulumi.Sprintf(`#!/bin/bash
set -e

# Container-Optimized OS has read-only root filesystem
# Use /home/chronos which is writable
export HOME=/home/chronos

# Helper function to get metadata
get_metadata() {
  curl -sf "http://metadata.google.internal/computeMetadata/v1/instance/attributes/$1" -H "Metadata-Flavor: Google"
}

# Read configuration from instance metadata
IMAGE=$(get_metadata "namazu-image")
SOURCE_TYPE=$(get_metadata "namazu-source-type")
SOURCE_ENDPOINT=$(get_metadata "namazu-source-endpoint")
API_ADDR=$(get_metadata "namazu-api-addr")
STORE_PROJECT_ID=$(get_metadata "namazu-store-project-id")
STORE_DATABASE=$(get_metadata "namazu-store-database")
DOMAIN=$(get_metadata "namazu-domain")
AUTH_TENANT_ID=$(get_metadata "namazu-auth-tenant-id")

# Configure docker credential helper for Artifact Registry
docker-credential-gcr configure-docker --registries=%s-docker.pkg.dev

# Pull the latest image
docker pull ${IMAGE}

# Stop and remove existing containers if they exist
docker stop namazu 2>/dev/null || true
docker rm namazu 2>/dev/null || true
docker stop caddy 2>/dev/null || true
docker rm caddy 2>/dev/null || true

# Create docker network for container communication
docker network create namazu-net 2>/dev/null || true

# Run the namazu container (internal only, not exposed to host)
docker run -d \
  --name namazu \
  --restart=always \
  --network namazu-net \
  -e NAMAZU_SOURCE_TYPE=${SOURCE_TYPE} \
  -e NAMAZU_SOURCE_ENDPOINT=${SOURCE_ENDPOINT} \
  -e NAMAZU_API_ADDR=${API_ADDR} \
  -e NAMAZU_STORE_PROJECT_ID=${STORE_PROJECT_ID} \
  -e NAMAZU_STORE_DATABASE=${STORE_DATABASE} \
  -e NAMAZU_AUTH_TENANT_ID=${AUTH_TENANT_ID} \
  ${IMAGE}

# Run Caddy for HTTPS reverse proxy (only if domain is set)
if [ -n "${DOMAIN}" ]; then
  docker run -d \
    --name caddy \
    --restart=always \
    --network namazu-net \
    -p 80:80 \
    -p 443:443 \
    -v /home/chronos/caddy_data:/data \
    caddy caddy reverse-proxy --from ${DOMAIN} --to namazu:8080
fi
`, region)

		instanceName := fmt.Sprintf("%s-instance", namePrefix)
		instance, err := compute.NewInstance(ctx, instanceName, &compute.InstanceArgs{
			Name:        pulumi.String(instanceName),
			MachineType: pulumi.String(machineType),
			Zone:        pulumi.String(zone),
			Tags:        pulumi.StringArray{pulumi.String("namazu-server")},
			BootDisk: &compute.InstanceBootDiskArgs{
				InitializeParams: &compute.InstanceBootDiskInitializeParamsArgs{
					Image: pulumi.String("cos-cloud/cos-stable"),
					Size:  pulumi.Int(10),
					Type:  pulumi.String("pd-standard"),
				},
			},
			NetworkInterfaces: compute.InstanceNetworkInterfaceArray{
				&compute.InstanceNetworkInterfaceArgs{
					Network:    network.ID(),
					Subnetwork: subnet.ID(),
					AccessConfigs: compute.InstanceNetworkInterfaceAccessConfigArray{
						&compute.InstanceNetworkInterfaceAccessConfigArgs{
							NatIp: staticIP.Address,
						},
					},
				},
			},
			ServiceAccount: &compute.InstanceServiceAccountArgs{
				Email: serviceAccount.Email,
				Scopes: pulumi.StringArray{
					pulumi.String("https://www.googleapis.com/auth/cloud-platform"),
				},
			},
			Metadata: pulumi.StringMap{
				"namazu-registry":         pulumi.Sprintf("%s-docker.pkg.dev/%s/namazu", region, project),
				"namazu-image":            pulumi.Sprintf("%s-docker.pkg.dev/%s/namazu/namazu:latest", region, project),
				"namazu-source-type":      pulumi.String("p2pquake"),
				"namazu-source-endpoint":  pulumi.String(sourceEndpoint),
				"namazu-api-addr":         pulumi.String(":8080"),
				"namazu-store-project-id": pulumi.String(project),
				"namazu-store-database":   pulumi.String(dbName),
				"namazu-domain":           pulumi.String(domain),
				"namazu-auth-tenant-id":   pulumi.String(authTenantId),
			},
			MetadataStartupScript:  startupScript,
			AllowStoppingForUpdate: pulumi.Bool(true),
			Description:            pulumi.String(fmt.Sprintf("Namazu %s server", env)),
		}, pulumi.DependsOn(iamBindings))
		if err != nil {
			return err
		}

		// =================================================================
		// Outputs
		// =================================================================
		ctx.Export("registryUrl", pulumi.Sprintf("%s-docker.pkg.dev/%s/namazu", region, project))
		ctx.Export("instanceName", instance.Name)
		ctx.Export("instanceZone", pulumi.String(zone))
		ctx.Export("externalIp", staticIP.Address)
		ctx.Export("serviceAccountEmail", pulumi.Sprintf("%s@%s.iam.gserviceaccount.com", saName, project))
		ctx.Export("firestoreDatabase", firestoreDB.Name)

		if domain != "" {
			ctx.Export("domain", pulumi.String(domain))
		}

		// Registry URL for pushing images
		ctx.Export("dockerPushCommand", pulumi.Sprintf(
			"docker push %s-docker.pkg.dev/%s/namazu/namazu:latest",
			region, project,
		))

		// SSH command via IAP
		ctx.Export("sshCommand", pulumi.Sprintf(
			"gcloud compute ssh %s --zone=%s --tunnel-through-iap",
			instance.Name, zone,
		))

		// Placeholder return to ensure registry is created
		_ = registry

		return nil
	})
}
