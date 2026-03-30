import { useState } from "react";
import { meRegenerateToken } from "@/api-client/sdk.gen";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

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
