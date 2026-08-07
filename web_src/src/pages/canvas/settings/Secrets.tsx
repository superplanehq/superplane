import { Loader2 } from "lucide-react";
import { useParams } from "react-router-dom";
import { useCanvas } from "@/hooks/useCanvasData";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { appSecretsPath } from "@/lib/appPaths";
import { cn } from "@/lib/utils";
import { Secrets as SecretsList } from "@/pages/organization/settings/Secrets";
import { PageHeader } from "./PageHeader";
import { SettingsTabs } from "./SettingsTabs";

/**
 * Canvas ("app")-scoped secrets list. Reuses the generic Secrets list
 * component from organization settings, scoped to the DOMAIN_TYPE_CANVAS
 * domain instead of the organization.
 */
export function CanvasSecretsPage() {
  const { organizationId = "", appId = "" } = useParams<{ organizationId: string; appId: string }>();
  const canvasId = appId;

  const { data: canvas, isLoading, error } = useCanvas(organizationId, canvasId);

  useReportPageReady(!isLoading, { failed: !!error });

  if (!organizationId || !canvasId) {
    return null;
  }

  const canvasName = canvas?.metadata?.name || "App";

  return (
    <div className={cn("flex h-full min-h-0 flex-col bg-slate-100", appDarkModeClasses.surface)}>
      <PageHeader organizationId={organizationId} title={`${canvasName} · Secrets`} />
      <SettingsTabs organizationId={organizationId} appId={canvasId} />
      <div className="min-h-0 flex-1 overflow-auto px-4">
        <div className="mx-auto w-full max-w-3xl">
          {isLoading ? (
            <div className="flex items-center justify-center py-24">
              <Loader2
                className={cn("h-8 w-8 animate-spin text-slate-400", appDarkModeClasses.textMuted)}
                aria-label="Loading"
              />
            </div>
          ) : (
            <SecretsList
              organizationId={organizationId}
              domainId={canvasId}
              domainType="DOMAIN_TYPE_CANVAS"
              basePath={appSecretsPath(organizationId, canvasId)}
            />
          )}
        </div>
      </div>
    </div>
  );
}
