import { useParams } from "react-router-dom";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { appSecretsPath } from "@/lib/appPaths";
import { cn } from "@/lib/utils";
import { SecretDetail as SecretDetailView } from "@/pages/organization/settings/SecretDetail";
import { PageHeader } from "./PageHeader";
import { SettingsTabs } from "./SettingsTabs";

/**
 * Canvas ("app")-scoped secret detail page. Reuses the generic SecretDetail
 * component from organization settings, scoped to the DOMAIN_TYPE_CANVAS
 * domain instead of the organization.
 */
export function CanvasSecretDetailPage() {
  const { organizationId = "", appId = "" } = useParams<{ organizationId: string; appId: string }>();
  const canvasId = appId;

  if (!organizationId || !canvasId) {
    return null;
  }

  return (
    <div className={cn("flex h-full min-h-0 flex-col bg-slate-100", appDarkModeClasses.surface)}>
      <PageHeader organizationId={organizationId} title="App · Secrets" />
      <SettingsTabs organizationId={organizationId} appId={canvasId} />
      <div className="min-h-0 flex-1 overflow-auto px-4">
        <div className="mx-auto w-full max-w-3xl">
          <SecretDetailView
            organizationId={organizationId}
            domainId={canvasId}
            domainType="DOMAIN_TYPE_CANVAS"
            basePath={appSecretsPath(organizationId, canvasId)}
          />
        </div>
      </div>
    </div>
  );
}
