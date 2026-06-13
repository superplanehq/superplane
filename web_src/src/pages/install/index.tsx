import AuthGuard from "@/components/AuthGuard";
import { usePageTitle } from "@/hooks/usePageTitle";
import { parseGitHubRepoParam } from "@/lib/githubRepo";
import { useMemo } from "react";
import { useSearchParams } from "react-router-dom";
import { InstallErrorView } from "./InstallErrorView";
import { InstallForm } from "./InstallForm";
import { InstallLoadingView } from "./InstallLoadingView";
import { InstallPageHeader } from "./InstallPageHeader";
import { InstallShell } from "./InstallShell";
import { useAutoInstall } from "./useAutoInstall";
import { useInstallApp } from "./useInstallApp";
import { useInstallPreview } from "./useInstallPreview";

export function InstallPage() {
  return (
    <AuthGuard>
      <InstallPageContent />
    </AuthGuard>
  );
}

function InstallPageContent() {
  usePageTitle(["Install App"]);

  const [searchParams] = useSearchParams();
  const repoParam = searchParams.get("repo");
  const presetName = searchParams.get("name")?.trim() || "";
  const presetOrganizationId = searchParams.get("organizationId")?.trim() || "";
  const parsedRepo = useMemo(() => parseGitHubRepoParam(repoParam), [repoParam]);

  const { preview, organizations, defaultName, defaultOrganizationId, loadError, isLoading } = useInstallPreview({
    repoParam,
    hasValidRepo: Boolean(parsedRepo),
    presetName,
    presetOrganizationId,
  });

  const isPreviewReady = Boolean(preview) && !isLoading && !loadError;

  const {
    name,
    setName,
    organizationId,
    setOrganizationId,
    nameError,
    isInstalling,
    handleInstall,
    clearNameError,
    showOrganizationPicker,
    installParamValues,
    handleInstallParamChange,
  } = useInstallApp({
    repoParam,
    defaultName,
    defaultOrganizationId,
    organizations,
    isPreviewReady,
    installParams: preview?.installParams,
  });

  const { canAutoInstall } = useAutoInstall({
    presetName,
    presetOrganizationId,
    preview,
    organizations,
    isLoading,
    isInstalling,
    onInstall: handleInstall,
  });

  if (isLoading || (canAutoInstall && isInstalling)) {
    return <InstallLoadingView message={canAutoInstall ? "Installing app..." : "Loading installation..."} />;
  }

  if (loadError || !preview) {
    return <InstallErrorView loadError={loadError} />;
  }

  return (
    <InstallShell>
      <InstallPageHeader title={preview.title} />

      <div className="rounded-lg bg-white p-6 shadow-sm outline outline-slate-950/10 dark:bg-gray-900 dark:outline-gray-800">
        <InstallForm
          name={name}
          nameError={nameError}
          organizationId={organizationId}
          organizations={organizations}
          showOrganizationPicker={showOrganizationPicker}
          isInstalling={isInstalling}
          installParams={preview?.installParams}
          installParamValues={installParamValues}
          onNameChange={(value) => {
            setName(value);
            if (nameError) {
              clearNameError();
            }
          }}
          onOrganizationChange={setOrganizationId}
          onInstallParamChange={handleInstallParamChange}
          onSubmit={() => {
            void handleInstall();
          }}
        />
      </div>
    </InstallShell>
  );
}
