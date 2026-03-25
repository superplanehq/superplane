import { useState } from "react";
import { meRegenerateToken } from "@/api-client/sdk.gen";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import { showErrorToast, showSuccessToast } from "@/utils/toast";

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

export function useConnectCommand(organizationId?: string) {
  const [connectCommand, setConnectCommand] = useState<string | null>(null);
  const [generating, setGenerating] = useState(false);

  const handleGenerateConnect = async () => {
    if (!organizationId) return;
    try {
      setGenerating(true);
      const response = await meRegenerateToken(withOrganizationHeader({ organizationId }));
      const token = response.data?.token;
      if (!token) {
        showErrorToast("Failed to generate API token");
        return;
      }
      const cmd = `superplane connect ${window.location.origin} ${token}`;
      setConnectCommand(cmd);
      await navigator.clipboard.writeText(cmd);
      showSuccessToast("Connect command copied to clipboard");
    } catch (err) {
      showErrorToast(err instanceof Error ? err.message : "Failed to generate token");
    } finally {
      setGenerating(false);
    }
  };

  return { connectCommand, generating, handleGenerateConnect };
}
