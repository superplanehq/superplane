import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { cn } from "@/lib/utils";
import { useConnectedIntegrations, useAvailableIntegrations } from "@/hooks/useIntegrations";
import { useParams } from "react-router-dom";
import { useCallback, useMemo } from "react";

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

type ResolvedIntegration = {
  integrationName: string;
  instanceName: string;
  instanceId: string;
  state: string;
  definitionLabel?: string;
};

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

  const resolved = useMemo(
    () =>
      isUUID
        ? resolveConnectedIntegration(integrationRef, connectedQuery.data)
        : resolveAvailableIntegration(integrationRef, availableQuery.data),
    [isUUID, integrationRef, connectedQuery.data, availableQuery.data],
  );

  const displayName =
    label ||
    resolved.instanceName ||
    resolved.definitionLabel ||
    formatIntegrationName(resolved.integrationName || integrationRef);

  const tone = integrationTone(resolved.state);
  const isVendorOnly = !isUUID;
  const integrationName = resolved.integrationName || integrationRef;

  const handleClick = useCallback(() => {
    window.dispatchEvent(
      new CustomEvent("agent:open-integration", {
        detail: {
          integrationName,
          instanceId: resolved.instanceId,
          instanceName: resolved.instanceName,
        },
      }),
    );
  }, [integrationName, resolved.instanceId, resolved.instanceName]);

  return (
    <button
      type="button"
      onClick={handleClick}
      className={cn(
        "inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md",
        "border bg-white",
        "text-xs font-medium",
        "transition-all cursor-pointer align-middle",
        toneClassName(tone),
      )}
      title={buttonTitle(tone, displayName)}
    >
      <IntegrationIcon integrationName={integrationName} className="h-4 w-4" size={16} />
      <span>{displayName}</span>
      {tone === "connected" && <span className="inline-block w-1.5 h-1.5 rounded-full bg-emerald-500 shrink-0" />}
      {tone === "pending" && <span className="inline-block w-1.5 h-1.5 rounded-full bg-amber-500 shrink-0" />}
      {tone === "error" && <span className="inline-block w-1.5 h-1.5 rounded-full bg-red-500 shrink-0" />}
      {isVendorOnly && <span className="text-[10px] leading-none text-slate-400 font-bold shrink-0">+</span>}
    </button>
  );
}

function resolveConnectedIntegration(
  integrationRef: string,
  connectedIntegrations?: Array<{
    metadata?: { id?: string; integrationName?: string; name?: string };
    status?: { state?: string };
  }>,
): ResolvedIntegration {
  const instance = connectedIntegrations?.find((item) => item.metadata?.id === integrationRef);
  if (!instance) {
    return { integrationName: "", instanceName: "", instanceId: integrationRef, state: "unknown" };
  }

  return {
    integrationName: instance.metadata?.integrationName ?? "",
    instanceName: instance.metadata?.name ?? "",
    instanceId: integrationRef,
    state: instance.status?.state ?? "unknown",
  };
}

function resolveAvailableIntegration(
  integrationRef: string,
  availableIntegrations?: Array<{ name?: string; label?: string }>,
): ResolvedIntegration {
  const definition = availableIntegrations?.find((item) => item.name === integrationRef);
  return {
    integrationName: integrationRef,
    instanceName: "",
    instanceId: "",
    state: "",
    definitionLabel: definition?.label,
  };
}

function integrationTone(state: string): "connected" | "pending" | "error" | "default" {
  switch (state) {
    case "ready":
      return "connected";
    case "pending":
      return "pending";
    case "error":
      return "error";
    default:
      return "default";
  }
}

function toneClassName(tone: "connected" | "pending" | "error" | "default"): string {
  switch (tone) {
    case "connected":
      return "border-emerald-200 text-emerald-700 hover:bg-emerald-50 hover:border-emerald-300";
    case "pending":
      return "border-amber-200 text-amber-700 hover:bg-amber-50 hover:border-amber-300";
    case "error":
      return "border-red-200 text-red-700 hover:bg-red-50 hover:border-red-300";
    default:
      return "border-slate-200 text-slate-700 hover:bg-slate-50 hover:border-slate-300";
  }
}

function buttonTitle(tone: "connected" | "pending" | "error" | "default", displayName: string): string {
  switch (tone) {
    case "connected":
      return `Connected: ${displayName}`;
    case "pending":
      return `Pending: ${displayName}`;
    case "error":
      return `Error: ${displayName}`;
    default:
      return `Connect ${displayName}`;
  }
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
