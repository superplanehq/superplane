import { useEffect, useState } from "react";
import { showErrorToast } from "@/lib/toast";
import type { InstallPreview, OrganizationOption } from "./types";

const INVALID_REPO_MESSAGE = "Invalid repository URL. Expected format: github.com/owner/repo";

interface UseInstallPreviewOptions {
  repoParam: string | null;
  hasValidRepo: boolean;
  presetName: string;
  presetOrganizationId: string;
}

export function useInstallPreview({
  repoParam,
  hasValidRepo,
  presetName,
  presetOrganizationId,
}: UseInstallPreviewOptions) {
  const [preview, setPreview] = useState<InstallPreview | null>(null);
  const [organizations, setOrganizations] = useState<OrganizationOption[]>([]);
  const [defaultName, setDefaultName] = useState("");
  const [defaultOrganizationId, setDefaultOrganizationId] = useState("");
  const [loadError, setLoadError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    if (!hasValidRepo || !repoParam) {
      setLoadError(INVALID_REPO_MESSAGE);

      setIsLoading(false);
      return;
    }

    const load = async () => {
      try {
        setIsLoading(true);
        setLoadError(null);

        const [previewResponse, organizationsResponse] = await Promise.all([
          fetch(`/apps/install/preview?repo=${encodeURIComponent(repoParam)}`, {
            credentials: "include",
          }),
          fetch("/organizations", {
            credentials: "include",
          }),
        ]);

        if (!previewResponse.ok) {
          const message = await previewResponse.text();
          throw new Error(message || "Failed to load app details");
        }

        if (!organizationsResponse.ok) {
          throw new Error("Failed to load organizations");
        }

        const previewData = (await previewResponse.json()) as InstallPreview;
        const organizationData = (await organizationsResponse.json()) as OrganizationOption[];

        setPreview(previewData);
        setOrganizations(organizationData);
        setDefaultName(presetName || previewData.defaultName);
        setDefaultOrganizationId(presetOrganizationId || (organizationData.length === 1 ? organizationData[0].id : ""));
      } catch (error) {
        const message = error instanceof Error ? error.message : "Failed to load installation details";
        setLoadError(message);
        showErrorToast(message);
      } finally {
        setIsLoading(false);
      }
    };

    void load();
  }, [hasValidRepo, presetName, presetOrganizationId, repoParam]);

  return {
    preview,
    organizations,
    defaultName,
    defaultOrganizationId,
    loadError,
    isLoading,
  };
}
