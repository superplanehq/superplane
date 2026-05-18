import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { cn } from "@/lib/utils";
import { useConnectedIntegrations, useAvailableIntegrations } from "@/hooks/useIntegrations";
import { useParams } from "react-router-dom";
import { useMemo } from "react";

interface IntegrationButtonProps {
  /**
   * Integration reference from the markdown link href.
   * Can be:
   *   - A UUID (instance ID): "791ee6d1-5399-47f0-aba2-08a8d2915cc9"
   *   - A vendor name: "github" (for connect-new flow)
   */
  integrationRef: string;
  /** Display label override (from markdown link text) */
  label?: string;
}

/**
 * Renders an integration reference as a clickable button with the vendor icon.
 * Resolves instance ID → name + icon from connected integrations.
 * Dispatches a CustomEvent so the parent page can open the integration dialog.
 *
 * Agent outputs:
 *   [Dash0](integration:791ee6d1-...)     -> resolved button for specific instance
 *   [GitHub](integration:github)           -> connect/create dialog for GitHub
 */
export function IntegrationButton({ integrationRef, label }: IntegrationButtonProps) {
  const { organizationId } = useParams<{ organizationId: string }>();
  const isUUID = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(integrationRef);

  const connectedQuery = useConnectedIntegrations(organizationId ?? "", { enabled: isUUID });
  const availableQuery = useAvailableIntegrations({ enabled: !isUUID });

  const resolved = useMemo(() => {
    if (isUUID) {
      const instance = connectedQuery.data?.find(
        (i: { metadata?: { id?: string } }) => i.metadata?.id === integrationRef,
      );
      if (instance) {
        return {
          integrationName: instance.metadata?.integrationName ?? "",
          instanceName: instance.metadata?.name ?? "",
          instanceId: integrationRef,
          state: instance.status?.state ?? "unknown",
        };
      }
      // UUID not found — still show a fallback
      return { integrationName: "", instanceName: "", instanceId: integrationRef, state: "unknown" };
    }

    // Vendor name reference — find the definition
    const def = availableQuery.data?.find((d: { name?: string }) => d.name === integrationRef);
    return {
      integrationName: integrationRef,
      instanceName: "",
      instanceId: "",
      state: "",
      definitionLabel: def?.label,
    };
  }, [isUUID, integrationRef, connectedQuery.data, availableQuery.data]);

  const displayName =
    label ||
    resolved.instanceName ||
    (resolved as { definitionLabel?: string }).definitionLabel ||
    formatIntegrationName(resolved.integrationName || integrationRef);

  const isConnected = resolved.state === "ready";
  const isPending = resolved.state === "pending";
  const isError = resolved.state === "error";
  const isVendorOnly = !isUUID;

  function handleClick() {
    window.dispatchEvent(
      new CustomEvent("agent:open-integration", {
        detail: {
          integrationName: resolved.integrationName || integrationRef,
          instanceId: resolved.instanceId,
          instanceName: resolved.instanceName,
        },
      }),
    );
  }

  return (
    <button
      type="button"
      onClick={handleClick}
      className={cn(
        "inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md",
        "border bg-white",
        "text-xs font-medium",
        "transition-all cursor-pointer align-middle",
        isConnected && "border-emerald-200 text-emerald-700 hover:bg-emerald-50 hover:border-emerald-300",
        isPending && "border-amber-200 text-amber-700 hover:bg-amber-50 hover:border-amber-300",
        isError && "border-red-200 text-red-700 hover:bg-red-50 hover:border-red-300",
        !isConnected &&
          !isPending &&
          !isError &&
          "border-slate-200 text-slate-700 hover:bg-slate-50 hover:border-slate-300",
      )}
      title={
        isConnected
          ? `Connected: ${displayName}`
          : isPending
            ? `Pending: ${displayName}`
            : isError
              ? `Error: ${displayName}`
              : `Connect ${displayName}`
      }
    >
      <IntegrationIcon integrationName={resolved.integrationName || integrationRef} className="h-4 w-4" size={16} />
      <span>{displayName}</span>
      {isConnected && <span className="inline-block w-1.5 h-1.5 rounded-full bg-emerald-500 shrink-0" />}
      {isPending && <span className="inline-block w-1.5 h-1.5 rounded-full bg-amber-500 shrink-0" />}
      {isError && <span className="inline-block w-1.5 h-1.5 rounded-full bg-red-500 shrink-0" />}
      {isVendorOnly && <span className="text-[10px] leading-none text-slate-400 font-bold shrink-0">+</span>}
    </button>
  );
}

/** Capitalize and clean up integration names for display */
function formatIntegrationName(name: string): string {
  return name
    .split(".")
    .map((part) => {
      if (part.length <= 3) return part.toUpperCase();
      return part.charAt(0).toUpperCase() + part.slice(1);
    })
    .join(" ");
}
