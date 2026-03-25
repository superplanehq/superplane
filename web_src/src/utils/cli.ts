export function detectPlatform(): string {
  const ua = navigator.userAgent.toLowerCase();
  const isLinux = ua.includes("linux");
  const isArm = ua.includes("arm") || ua.includes("aarch64");
  const os = isLinux ? "linux" : "darwin";
  const arch = isArm ? "arm64" : "amd64";
  return `${os}-${arch}`;
}
