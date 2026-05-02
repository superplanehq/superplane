import type { IntegrationsIntegrationDefinition, OrganizationsIntegration } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { CopyButton } from "@/ui/CopyButton";
import { ArrowLeft, Plug, Trash2 } from "lucide-react";
import { Link } from "react-router-dom";

export interface HeaderProps {
  organizationId: string;
  integration: OrganizationsIntegration;
  integrationDef?: IntegrationsIntegrationDefinition;
  canDeleteIntegrations: boolean;
  permissionsLoading: boolean;
  onRequestDelete: () => void;
}

export function Header({
  organizationId,
  integration,
  integrationDef,
  canDeleteIntegrations,
  permissionsLoading,
  onRequestDelete,
}: HeaderProps) {
  const integrationsHref = `/${organizationId}/settings/integrations`;
  const integrationId = integration.metadata?.id;
  const integrationName = integration.metadata?.name;
  const integrationStatus = integration.status?.state || "unknown";

  return (
    <div className="flex flex-wrap items-center gap-4 mb-6">
      <Link
        to={integrationsHref}
        className="text-gray-500 dark:text-gray-400 hover:text-gray-800 dark:hover:text-gray-100"
        aria-label="Back to integrations"
      >
        <ArrowLeft className="w-5 h-5" />
      </Link>
      <IntegrationIcon
        integrationName={integration?.metadata?.integrationName}
        iconSlug={integrationDef?.icon}
        className="w-6 h-6"
      />
      <div className="flex-1 min-w-[200px]">
        <h4 className="flex items-center text-2xl font-medium">
          <span
            className="inline-flex shrink-0"
            title={integrationStatus.charAt(0).toUpperCase() + integrationStatus.slice(1)}
          ></span>
          <span>{integrationName}</span>
        </h4>
        {integrationId ? (
          <div className="mt-1.5 flex max-w-full items-center gap-1.5">
            <span className="min-w-0 truncate font-mono text-xs text-gray-700 dark:text-gray-300">{integrationId}</span>
            <CopyButton text={integrationId} />
          </div>
        ) : null}
      </div>
      <div className="ml-auto flex items-center gap-2">
        <Plug
          className={`h-5 w-5 ${
            integrationStatus === "ready"
              ? "text-green-500"
              : integrationStatus === "error"
                ? "text-red-600"
                : "text-amber-600"
          }`}
          aria-label={`Integration status: ${integrationStatus}`}
        />
        <PermissionTooltip
          allowed={canDeleteIntegrations || permissionsLoading}
          message="You don't have permission to delete integrations."
        >
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            className="shrink-0 text-gray-500 hover:text-red-600 dark:text-gray-400 dark:hover:text-red-400"
            aria-label="Delete integration"
            disabled={!canDeleteIntegrations}
            onClick={() => {
              if (!canDeleteIntegrations) return;
              onRequestDelete();
            }}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </PermissionTooltip>
      </div>
    </div>
  );
}
