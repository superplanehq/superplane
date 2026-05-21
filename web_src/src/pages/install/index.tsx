import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import AuthGuard from "@/components/AuthGuard";
import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { parseGitHubRepoParam } from "@/lib/githubRepo";
import { showErrorToast } from "@/lib/toast";
import { getUsageLimitToastMessage } from "@/lib/usageLimits";

const MAX_APP_NAME_LENGTH = 50;

interface OrganizationOption {
  id: string;
  name: string;
}

interface InstallPreview {
  repo: string;
  title: string;
  description?: string;
  defaultName: string;
}

interface InstallResult {
  canvasId: string;
  organizationId: string;
}

export function InstallPage() {
  return (
    <AuthGuard>
      <InstallPageContent />
    </AuthGuard>
  );
}

function InstallPageContent() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const repoParam = searchParams.get("repo");
  const presetName = searchParams.get("name")?.trim() || "";
  const presetOrganizationId = searchParams.get("organizationId")?.trim() || "";

  const parsedRepo = useMemo(() => parseGitHubRepoParam(repoParam), [repoParam]);
  const [preview, setPreview] = useState<InstallPreview | null>(null);
  const [organizations, setOrganizations] = useState<OrganizationOption[]>([]);
  const [name, setName] = useState("");
  const [organizationId, setOrganizationId] = useState("");
  const [nameError, setNameError] = useState("");
  const [loadError, setLoadError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isInstalling, setIsInstalling] = useState(false);
  const autoInstallAttempted = useRef(false);

  usePageTitle(["Install App"]);

  useEffect(() => {
    if (!parsedRepo || !repoParam) {
      setLoadError("A valid GitHub repository is required. Use ?repo=github.com/owner/repository");
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
        setName(presetName || previewData.defaultName);

        const selectedOrganizationId =
          presetOrganizationId || (organizationData.length === 1 ? organizationData[0].id : "");
        setOrganizationId(selectedOrganizationId);
      } catch (error) {
        const message = error instanceof Error ? error.message : "Failed to load installation details";
        setLoadError(message);
        showErrorToast(message);
      } finally {
        setIsLoading(false);
      }
    };

    void load();
  }, [parsedRepo, presetName, presetOrganizationId, repoParam]);

  const handleInstall = useCallback(
    async (installName = name, installOrganizationId = organizationId) => {
      const trimmedName = installName.trim();

      if (!trimmedName) {
        setNameError("Name is required");
        return;
      }

      if (trimmedName.length > MAX_APP_NAME_LENGTH) {
        setNameError(`Name must be ${MAX_APP_NAME_LENGTH} characters or less`);
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
    void handleInstall(presetName, presetOrganizationId);
  }, [canAutoInstall, handleInstall, isInstalling, isLoading, presetName, presetOrganizationId]);

  if (isLoading || (canAutoInstall && isInstalling)) {
    return (
      <InstallShell>
        <div className="flex items-center justify-center py-16">
          <div className="animate-spin rounded-full h-8 w-8 border-b border-blue-600" />
          <p className="ml-3 text-gray-500">{canAutoInstall ? "Installing app..." : "Loading installation..."}</p>
        </div>
      </InstallShell>
    );
  }

  if (loadError || !preview) {
    return (
      <InstallShell>
        <div className="rounded-md border border-red-200 bg-red-50 p-4 text-sm text-red-700">
          {loadError || "Unable to load app installation details."}
        </div>
      </InstallShell>
    );
  }

  const showOrganizationPicker = organizations.length > 1;

  return (
    <InstallShell>
      <div className="mx-auto w-full max-w-lg">
        <Heading level={2} className="!text-2xl mb-2">
          {preview.title}
        </Heading>
        {preview.description ? (
          <Text className="text-gray-600 dark:text-gray-400 mb-8">{preview.description}</Text>
        ) : (
          <div className="mb-8" />
        )}

        <form
          className="space-y-6"
          onSubmit={(event) => {
            event.preventDefault();
            void handleInstall();
          }}
        >
          <div className="space-y-2">
            <Label htmlFor="install-app-name">Name</Label>
            <Input
              id="install-app-name"
              data-testid="install-app-name-input"
              value={name}
              maxLength={MAX_APP_NAME_LENGTH}
              onChange={(event) => {
                setName(event.target.value);
                if (nameError) {
                  setNameError("");
                }
              }}
            />
            {nameError ? <p className="text-xs text-red-600">{nameError}</p> : null}
          </div>

          {showOrganizationPicker ? (
            <div className="space-y-2">
              <Label htmlFor="install-app-organization">
                Organization to which you want to install if you have more than one
              </Label>
              <Select value={organizationId} onValueChange={setOrganizationId}>
                <SelectTrigger id="install-app-organization" data-testid="install-app-organization-select">
                  <SelectValue placeholder="Select an organization" />
                </SelectTrigger>
                <SelectContent>
                  {organizations.map((organization) => (
                    <SelectItem key={organization.id} value={organization.id}>
                      {organization.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          ) : null}

          <LoadingButton
            type="submit"
            data-testid="install-app-submit"
            loading={isInstalling}
            loadingText="Installing..."
            disabled={!name.trim() || !organizationId}
            className="w-full sm:w-auto"
          >
            Install
          </LoadingButton>
        </form>
      </div>
    </InstallShell>
  );
}

function InstallShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-slate-100 dark:bg-slate-900">
      <header className="bg-white border-b border-slate-950/15 px-4 h-12 flex items-center">
        <span className="text-sm font-medium text-slate-700">SuperPlane</span>
      </header>
      <main className="p-8">{children}</main>
    </div>
  );
}
