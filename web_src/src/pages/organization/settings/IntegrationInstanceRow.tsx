import { Plug } from "lucide-react";
import { useNavigate } from "react-router";
import type { OrganizationsIntegration } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface IntegrationInstanceRowProps {
  integration: OrganizationsIntegration;
  index: number;
  organizationId: string;
  canUpdateIntegrations: boolean;
  permissionsLoading: boolean;
}

export function IntegrationInstanceRow({
  integration,
  index,
  organizationId,
  canUpdateIntegrations,
  permissionsLoading,
}: IntegrationInstanceRowProps) {
  const navigate = useNavigate();
  const integrationDisplayName = integration.metadata?.name;
  const statusState = integration.status?.state;
  const statusLabel = statusState ? statusState.charAt(0).toUpperCase() + statusState.slice(1) : "Unknown";

  return (
    <div
      className={`flex items-center gap-2 py-1.5 border-t border-gray-200 dark:border-gray-700/70 ${index === 0 ? "mt-1" : ""}`}
    >
      <Plug
        className={`w-4 h-4 shrink-0 ${
          statusState === "ready" ? "text-green-500" : statusState === "error" ? "text-red-500" : "text-amber-600"
        }`}
      />
      <span
        className={cn(
          "inline-flex min-w-16 items-center justify-center rounded px-2 py-0.5 text-xs font-medium",
          statusState === "ready"
            ? "bg-white text-green-500 dark:bg-green-300 dark:text-green-950"
            : statusState === "error"
              ? "bg-white text-red-500 dark:bg-red-300 dark:text-red-950"
              : "bg-white text-amber-600 dark:bg-amber-300 dark:text-amber-950",
        )}
      >
        {statusLabel}
      </span>
      <p className="text-sm font-medium text-gray-800 dark:text-gray-100 truncate">{integrationDisplayName}</p>
      <div className="ml-auto flex items-center gap-4">
        <PermissionTooltip
          allowed={canUpdateIntegrations || permissionsLoading}
          message="You don't have permission to update integrations."
        >
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              if (!canUpdateIntegrations) return;
              const providerName = integration.metadata?.integrationName;
              if (providerName && integration.status?.setupState?.currentStep) {
                navigate(`/${organizationId}/settings/integrations/${providerName}/setup`, {
                  state: { integrationId: integration.metadata?.id },
                });
                return;
              }

              navigate(`/${organizationId}/settings/integrations/${integration.metadata?.id}`, {
                state: { tab: "configuration" },
              });
            }}
            disabled={!canUpdateIntegrations}
          >
            Configure
          </Button>
        </PermissionTooltip>
      </div>
    </div>
  );
}
