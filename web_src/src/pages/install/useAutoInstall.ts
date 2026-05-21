import { useEffect, useRef } from "react";
import type { InstallPreview, OrganizationOption } from "./types";

interface UseAutoInstallOptions {
  presetName: string;
  presetOrganizationId: string;
  preview: InstallPreview | null;
  organizations: OrganizationOption[];
  isLoading: boolean;
  isInstalling: boolean;
  onInstall: (name: string, organizationId: string) => Promise<void>;
}

export function useAutoInstall({
  presetName,
  presetOrganizationId,
  preview,
  organizations,
  isLoading,
  isInstalling,
  onInstall,
}: UseAutoInstallOptions) {
  const autoInstallAttempted = useRef(false);

  const canAutoInstall =
    Boolean(presetName) &&
    Boolean(presetOrganizationId) &&
    Boolean(preview) &&
    organizations.some((organization) => organization.id === presetOrganizationId);

  useEffect(() => {
    if (!canAutoInstall || autoInstallAttempted.current || isLoading || isInstalling) {
      return;
    }

    autoInstallAttempted.current = true;
    void onInstall(presetName, presetOrganizationId);
  }, [canAutoInstall, isInstalling, isLoading, onInstall, presetName, presetOrganizationId]);

  return { canAutoInstall };
}
