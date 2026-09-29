import { ExternalLink, Plug } from "lucide-react";
import { Button } from "@/components/ui/button";
import { PermissionTooltip } from "@/components/PermissionGate";
import { githubAppInstallationUrl } from "@/lib/integrations";
import { integrationStatusLabel, type IntegrationCatalogItem } from "@/lib/integrationCatalog";
import {
  instancePlugClass,
  instanceStatusLabelClass,
  type CatalogAppearance,
  type catalogAppearance,
} from "./integrationCatalogAppearance";

type CatalogStyles = ReturnType<typeof catalogAppearance>;

/** A single connected integration instance, shown under its provider's card. */
export function CatalogInstanceRow({
  appearance,
  integration,
  styles,
  canUpdateIntegrations,
  permissionsLoading,
  onConfigure,
}: {
  appearance: CatalogAppearance;
  integration: IntegrationCatalogItem["instances"][number];
  styles: CatalogStyles;
  canUpdateIntegrations: boolean;
  permissionsLoading: boolean;
  onConfigure: () => void;
}) {
  const state = integration.status?.state;
  const installationUrl = githubAppInstallationUrl(integration);

  return (
    <div className={styles.instanceRow}>
      <Plug className={`size-4 shrink-0 ${instancePlugClass(state, styles)}`} />
      <span className={instanceStatusLabelClass(appearance, state, styles)}>{integrationStatusLabel(state)}</span>
      <p className={styles.instanceName}>{integration.metadata?.name}</p>
      <div className="ml-auto flex items-center gap-2">
        {installationUrl ? (
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => window.open(installationUrl, "_blank", "noopener,noreferrer")}
          >
            Manage installation
            <ExternalLink className="size-3.5" aria-hidden />
          </Button>
        ) : null}
        <PermissionTooltip
          allowed={canUpdateIntegrations || permissionsLoading}
          message="You don't have permission to update integrations."
        >
          <Button variant="outline" size="sm" onClick={onConfigure} disabled={!canUpdateIntegrations}>
            Configure
          </Button>
        </PermissionTooltip>
      </div>
    </div>
  );
}
