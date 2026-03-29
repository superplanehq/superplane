export function detectPlatform(): string {
  const ua = navigator.userAgent.toLowerCase();
  const isLinux = ua.includes("linux");
  const isArm = ua.includes("arm") || ua.includes("aarch64");
  const os = isLinux ? "linux" : "darwin";
  const arch = isArm ? "arm64" : "amd64";
  return `${os}-${arch}`;
}

export function getInstallCommand(platform: string): string {
  return `curl -L https://install.superplane.com/superplane-cli-${platform} -o superplane && chmod +x superplane && sudo mv superplane /usr/local/bin/`;
}
