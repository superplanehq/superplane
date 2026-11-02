import type { OrganizationsCreateFleetRunnerResponse, SuperplaneOrganizationsFleet } from "@/api-client";

const shellQuote = (value: string) => `'${value.replace(/'/g, "'\\''")}'`;

function publishedRelease(fleet: SuperplaneOrganizationsFleet) {
  const { operatingSystem = "linux", architecture = "amd64" } = fleet.spec ?? {};
  const version = fleet.runnerVersion ?? "";
  if (
    !/^v\d+\.\d+\.\d+(?:[-+][\w.-]+)?$/.test(version) ||
    operatingSystem !== "linux" ||
    !["amd64", "arm64"].includes(architecture)
  )
    return null;
  return { version, architecture };
}

export function runnerDownloadCommand(fleet: SuperplaneOrganizationsFleet): string | null {
  const release = publishedRelease(fleet);
  if (!release) return null;
  const url = `https://github.com/superplanehq/superplane/releases/download/${encodeURIComponent(`runner/${release.version}`)}/runner-linux-${release.architecture}.tar.gz`;
  return `curl -fsSL ${shellQuote(url)} -o runner.tar.gz
tar -xzf runner.tar.gz runner`;
}

export function defaultRunnerImage(fleet: SuperplaneOrganizationsFleet): string {
  if (fleet.runnerVersion === "dev") return "superplane-runner-local:dev";
  const release = publishedRelease(fleet);
  return release ? `ghcr.io/superplanehq/runner:${release.version}` : "";
}

export function runnerCommand(
  registration: OrganizationsCreateFleetRunnerResponse,
  method: string,
  image: string,
  fleet: SuperplaneOrganizationsFleet,
): string {
  const token = shellQuote(registration.registrationToken!);
  if (method === "binary")
    return `./runner --url ${shellQuote(registration.runnerApiUrl!)} --registration-token ${token}`;
  const url = new URL(registration.runnerApiUrl!);
  const loopback = ["localhost", "127.0.0.1", "[::1]"].includes(url.hostname);
  if (loopback) url.hostname = "host.docker.internal";
  const platform = `${fleet.spec?.operatingSystem || "linux"}/${fleet.spec?.architecture || "amd64"}`;
  return [
    "docker run --rm -it",
    `  --platform ${shellQuote(platform)}`,
    ...(loopback ? ["  --add-host host.docker.internal:host-gateway"] : []),
    `  ${shellQuote(image)}`,
    `  --url ${shellQuote(url.toString().replace(/\/$/, ""))}`,
    `  --registration-token ${token}`,
  ].join(" \\\n");
}
