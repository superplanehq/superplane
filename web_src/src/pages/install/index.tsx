import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { UsageLimitAlert } from "@/components/UsageLimitAlert";
import { Heading } from "@/components/Heading/heading";
import { Text } from "@/components/Text/text";
import { Alert, AlertDescription, AlertTitle } from "@/ui/alert";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import AuthGuard from "@/components/AuthGuard";
import { usePageTitle } from "@/hooks/usePageTitle";
import { getApiErrorMessage } from "@/lib/errors";
import { parseGitHubRepoParam } from "@/lib/githubRepo";
import { showErrorToast } from "@/lib/toast";
import { getUsageLimitNotice, getUsageLimitToastMessage } from "@/lib/usageLimits";
import { ExternalLink } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

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
        <div className="flex flex-col items-center justify-center gap-4 py-16">
          <div className="animate-spin rounded-full h-8 w-8 border-b border-blue-600" />
          <Text className="text-gray-500 dark:text-gray-400">
            {canAutoInstall ? "Installing app..." : "Loading installation..."}
          </Text>
        </div>
      </InstallShell>
    );
  }

  if (loadError || !preview) {
    const usageLimitNotice = loadError ? getUsageLimitNotice(loadError) : null;

    return (
      <InstallShell>
        <InstallPageHeader title="Install App" description="Add a pre-built app from GitHub to your organization." />
        <div className="rounded-lg bg-white p-6 shadow-sm outline outline-slate-950/10 dark:bg-gray-900 dark:outline-gray-800">
          {usageLimitNotice ? (
            <UsageLimitAlert notice={usageLimitNotice} />
          ) : (
            <Alert variant="destructive">
              <AlertTitle>Unable to install app</AlertTitle>
              <AlertDescription>{loadError || "Unable to load app installation details."}</AlertDescription>
            </Alert>
          )}
        </div>
      </InstallShell>
    );
  }

  const showOrganizationPicker = organizations.length > 1;
  const repoHref = preview.repo ? `https://${preview.repo.replace(/^https?:\/\//, "")}` : null;

  return (
    <InstallShell>
      <InstallPageHeader title={preview.title} />

      <div className="rounded-lg bg-white p-6 shadow-sm outline outline-slate-950/10 dark:bg-gray-900 dark:outline-gray-800">
        {repoHref ? (
          <div className="mb-6">
            <Text className="text-sm text-gray-500 dark:text-gray-400">
              Source repository{" "}
              <a
                href={repoHref}
                target="_blank"
                rel="noreferrer"
                className="inline-flex items-center gap-1 font-medium text-gray-800 hover:underline dark:text-gray-200"
              >
                {preview.repo}
                <ExternalLink className="h-3.5 w-3.5" aria-hidden />
              </a>
            </Text>
          </div>
        ) : null}

        <form
          className="space-y-6"
          onSubmit={(event) => {
            event.preventDefault();
            void handleInstall();
          }}
        >
          <div className="space-y-2">
            <Label htmlFor="install-app-name">App name</Label>
            <Input
              id="install-app-name"
              data-testid="install-app-name-input"
              value={name}
              maxLength={MAX_APP_NAME_LENGTH}
              autoFocus
              onChange={(event) => {
                if (event.target.value.length <= MAX_APP_NAME_LENGTH) {
                  setName(event.target.value);
                }

                if (nameError) {
                  setNameError("");
                }
              }}
              onKeyDown={(event) => {
                if (event.key === "Enter" && !event.shiftKey) {
                  event.preventDefault();
                  void handleInstall();
                }
              }}
            />
            {nameError ? <p className="text-xs text-red-600">{nameError}</p> : null}
          </div>

          {showOrganizationPicker ? (
            <div className="space-y-2">
              <Label htmlFor="install-app-organization">Organization</Label>
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

          <div className="flex flex-row justify-start gap-3 pt-2">
            <LoadingButton
              type="submit"
              data-testid="install-app-submit"
              loading={isInstalling}
              loadingText="Installing..."
              disabled={!name.trim() || !organizationId}
            >
              Install
            </LoadingButton>
          </div>
        </form>
      </div>
    </InstallShell>
  );
}

function InstallPageHeader({ title, description }: { title: string; description?: string }) {
  return (
    <div className="mb-6">
      <Heading level={2} className="!text-2xl mb-1">
        {title}
      </Heading>
      {description ? <Text className="text-gray-800 dark:text-gray-400">{description}</Text> : null}
    </div>
  );
}

function InstallShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen flex flex-col bg-slate-100 dark:bg-slate-900">
      <header className="flex h-12 items-center border-b border-slate-950/15 bg-white px-4 dark:border-gray-800 dark:bg-gray-900">
        <OrganizationMenuButton />
      </header>
      <main className="flex w-full flex-grow-1 flex-col">
        <div className="mx-auto w-full max-w-[640px] flex-grow-1 p-8">{children}</div>
      </main>
    </div>
  );
}
