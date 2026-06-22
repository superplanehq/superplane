import AuthGuard from "@/components/AuthGuard";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { parseGitHubRepoParam } from "@/lib/githubRepo";
import { useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { InstallErrorView } from "./InstallErrorView";
import { InstallLoadingView } from "./InstallLoadingView";
import { InstallShell } from "./InstallShell";
import { useInstallPreview } from "./useInstallPreview";
import { InstallProgressPanel } from "@/pages/home/InstallProgressPanel";
import type { AppEntry } from "@/pages/home/AppDetailModal";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Label } from "@/components/ui/label";

// Treat a cleared/whitespace-only override as "use the default" so the user
// still installs under the pre-filled name they saw rather than a randomly
// generated one on the server.
function resolveCanvasName(nameOverride: string | null, defaultName: string): string {
  return nameOverride?.trim() ? nameOverride : defaultName;
}

export function InstallPage() {
  return (
    <AuthGuard>
      <InstallPageContent />
    </AuthGuard>
  );
}

function InstallPageContent() {
  usePageTitle(["Install App"]);

  const navigate = useNavigate();
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

  const [organizationId, setOrganizationId] = useState("");
  const [nameOverride, setNameOverride] = useState<string | null>(null);

  // Set org once loaded
  const effectiveOrgId = organizationId || defaultOrganizationId;
  const showOrgPicker = organizations.length > 1;
  const canvasName = resolveCanvasName(nameOverride, defaultName);

  useReportPageReady(!isLoading, {
    failed: !!(loadError || !preview || !repoParam),
  });

  if (isLoading) {
    return <InstallLoadingView message="Loading installation..." />;
  }

  if (loadError || !preview || !repoParam) {
    return <InstallErrorView loadError={loadError} />;
  }

  // Build an AppEntry from the preview response (no manifest data for external repos)
  const app: AppEntry = {
    repo: repoParam,
    icon: "",
    title: preview.canvasName || preview.title || repoParam,
    description: preview.description || "",
    integrations: preview.integrations || [],
    tags: [],
    requirements: [],
    agentInstructions: "",
  };

  return (
    <InstallShell>
      {showOrgPicker && (
        <div className="mb-4 max-w-md">
          <Label className="text-xs mb-1.5">Organization</Label>
          <Select value={effectiveOrgId} onValueChange={setOrganizationId}>
            <SelectTrigger className="h-8 text-xs">
              <SelectValue placeholder="Select an organization" />
            </SelectTrigger>
            <SelectContent>
              {organizations.map((org) => (
                <SelectItem key={org.id} value={org.id}>
                  {org.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      )}

      {effectiveOrgId ? (
        <InstallProgressPanel
          app={app}
          organizationId={effectiveOrgId}
          canvasName={canvasName}
          onCanvasNameChange={setNameOverride}
          skipPreviewFetch
          preloadedIntegrations={preview.integrations}
          preloadedParams={preview.installParams}
          onClose={() => navigate(`/${effectiveOrgId}`)}
        />
      ) : (
        <p className="text-sm text-slate-500">Select an organization to continue.</p>
      )}
    </InstallShell>
  );
}
