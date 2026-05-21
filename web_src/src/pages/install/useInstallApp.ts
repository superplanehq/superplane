import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";
import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { MAX_APP_NAME_LENGTH } from "./constants";
import type { InstallResult, OrganizationOption } from "./types";

interface UseInstallAppOptions {
  repoParam: string | null;
  defaultName: string;
  defaultOrganizationId: string;
  organizations: OrganizationOption[];
  isPreviewReady: boolean;
}

export function useInstallApp({
  repoParam,
  defaultName,
  defaultOrganizationId,
  organizations,
  isPreviewReady,
}: UseInstallAppOptions) {
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [organizationId, setOrganizationId] = useState("");
  const [nameError, setNameError] = useState("");
  const [isInstalling, setIsInstalling] = useState(false);

  useEffect(() => {
    if (!isPreviewReady) {
      return;
    }

    setName(defaultName);
    setOrganizationId(defaultOrganizationId);
    setNameError("");
  }, [defaultName, defaultOrganizationId, isPreviewReady]);

  const handleInstall = useCallback(
    async (installName = name, installOrganizationId = organizationId) => {
      const trimmedName = installName.trim();
      const nameError = validateName(trimmedName);

      if (nameError) {
        setNameError(nameError);
        return;
      }

      if (!installOrganizationId) {
        showErrorToast("Select an organization to install this app into");
        return;
      }

      if (!repoParam) {
        return;
      }

      setNameError("");
      setIsInstalling(true);

      try {
        const response = await fetch("/apps/install", {
          method: "POST",
          credentials: "include",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            repo: repoParam,
            name: trimmedName,
            organizationId: installOrganizationId,
          }),
        });

        if (!response.ok) {
          const message = await response.text();
          throw new Error(message || "Failed to install app");
        }

        const result = (await response.json()) as InstallResult;
        navigate(`/${result.organizationId}/canvases/${result.canvasId}`);
      } catch (error) {
        const message = getUsageLimitToastMessage(error, getApiErrorMessage(error, "Failed to install app"));
        showErrorToast(message);

        if (message.toLowerCase().includes("already") || message.toLowerCase().includes("exists")) {
          setNameError("An app with this name already exists");
        }
      } finally {
        setIsInstalling(false);
      }
    },
    [name, navigate, organizationId, repoParam],
  );

  const clearNameError = useCallback(() => {
    setNameError("");
  }, []);

  return {
    name,
    setName,
    organizationId,
    setOrganizationId,
    nameError,
    isInstalling,
    handleInstall,
    clearNameError,
    showOrganizationPicker: organizations.length > 1,
  };
}

function validateName(name: string) {
  if (!name) {
    return "Name is required";
  }

  if (name.length > MAX_APP_NAME_LENGTH) {
    return `Name must be ${MAX_APP_NAME_LENGTH} characters or less`;
  }

  return null;
}
