export const CLI_INSTALL_COMMAND = "curl -fsSL https://install.superplane.com/install.sh | sh";

export const SUPPORTED_CLI_PLATFORMS = ["darwin-arm64", "darwin-amd64", "linux-arm64", "linux-amd64"] as const;

export type CliPlatform = (typeof SUPPORTED_CLI_PLATFORMS)[number];

export function getInstallCommand(): string {
  return CLI_INSTALL_COMMAND;
}

export function getCliBinaryURL(platform: CliPlatform, version?: string): string {
  const versionPath = version ? `/${version}` : "";
  return `https://install.superplane.com${versionPath}/superplane-cli-${platform}`;
}

export function getManualInstallCommand(platform: CliPlatform, version?: string): string {
  return `curl -L ${getCliBinaryURL(platform, version)} -o superplane && chmod +x superplane && sudo mv superplane /usr/local/bin/superplane`;
}
