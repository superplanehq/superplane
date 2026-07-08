import type { IntegrationCapabilityStateState } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { CapabilityGroupSection } from "@/lib/capabilities";
import { cn } from "@/lib/utils";
import { CopyButton } from "@/ui/CopyButton";
import { Check, CircleOff, Info, Minus, Plus } from "lucide-react";
import {
  DEFAULT_CAPABILITY_STATE,
  type DisplayCapability,
  getCapabilityDescription,
  getCapabilityStatusBadgeClassName,
  getCapabilityStatusBadgeDotClassName,
  getCapabilityStatusLabel,
} from "./lib";

export interface CapabilitySectionProps {
  section: CapabilityGroupSection;
  capabilityByName: Map<string, DisplayCapability>;
  capabilityStates: Record<string, IntegrationCapabilityStateState>;
  canUpdateIntegrations: boolean;
  permissionsLoading: boolean;
  capabilitiesMutationPending: boolean;
  onQueueCapabilityStateChange: (capability: DisplayCapability, nextState: IntegrationCapabilityStateState) => void;
}

type CapabilityStatusCellProps = {
  capability: DisplayCapability;
  effectiveState: IntegrationCapabilityStateState;
  description: string | undefined;
};

function CapabilityStatusCell({ capability, effectiveState, description }: CapabilityStatusCellProps) {
  return (
    <td className="min-w-0 px-4 py-3 align-middle">
      <div className="grid w-full grid-cols-[6.25rem_1.5rem_minmax(0,1fr)] items-center gap-x-2 gap-y-2">
        <Badge
          variant="outline"
          className={cn(
            "inline-flex w-fit max-w-full min-w-0 justify-self-start gap-1 font-medium",
            getCapabilityStatusBadgeClassName(effectiveState),
          )}
        >
          <span
            className={cn("size-2 shrink-0 rounded-full", getCapabilityStatusBadgeDotClassName(effectiveState))}
            aria-hidden
          />
          <span className="min-w-0 truncate">{getCapabilityStatusLabel(effectiveState)}</span>
        </Badge>
        <div className="flex w-6 shrink-0 items-center justify-center">
          {description ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
                  aria-label={`Description for ${capability.name}`}
                >
                  <Info className="size-4 shrink-0" aria-hidden />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="top" className="max-w-xs text-balance">
                {description}
              </TooltipContent>
            </Tooltip>
          ) : null}
        </div>
        <div className="flex min-w-0 flex-wrap items-center gap-x-2 gap-y-1 font-mono text-sm text-gray-800 dark:text-gray-100">
          <span className="min-w-0 break-all">{capability.name}</span>
          <CopyButton text={capability.name} />
        </div>
      </div>
    </td>
  );
}

type CapabilityActionsCellProps = {
  capability: DisplayCapability;
  effectiveState: IntegrationCapabilityStateState;
  serverState: IntegrationCapabilityStateState;
  isDirty: boolean;
  actionDisabled: boolean;
  canUpdateIntegrations: boolean;
  permissionsLoading: boolean;
  onQueueCapabilityStateChange: (capability: DisplayCapability, nextState: IntegrationCapabilityStateState) => void;
};

function CapabilityActionsCell({
  capability,
  effectiveState,
  serverState,
  isDirty,
  actionDisabled,
  canUpdateIntegrations,
  permissionsLoading,
  onQueueCapabilityStateChange,
}: CapabilityActionsCellProps) {
  return (
    <td className="min-w-0 px-4 py-3 align-middle text-right whitespace-nowrap">
      <PermissionTooltip
        allowed={canUpdateIntegrations || permissionsLoading}
        message="You don't have permission to update integrations."
      >
        <span className="flex justify-end gap-2">
          {effectiveState === "STATE_ENABLED" ? (
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={actionDisabled}
              onClick={() => onQueueCapabilityStateChange(capability, "STATE_DISABLED")}
            >
              <CircleOff className="size-4 shrink-0" aria-hidden />
              Disable
            </Button>
          ) : null}
          {effectiveState === "STATE_DISABLED" ? (
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={actionDisabled}
              onClick={() => onQueueCapabilityStateChange(capability, "STATE_ENABLED")}
            >
              <Check className="size-4 shrink-0" aria-hidden />
              Enable
            </Button>
          ) : null}
          {effectiveState === "STATE_AVAILABLE" ? (
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={actionDisabled}
              onClick={() => onQueueCapabilityStateChange(capability, "STATE_REQUESTED")}
            >
              <Plus className="size-4 shrink-0" aria-hidden />
              Add
            </Button>
          ) : null}
          {effectiveState === "STATE_REQUESTED" && isDirty ? (
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={actionDisabled}
              onClick={() => onQueueCapabilityStateChange(capability, serverState)}
            >
              <Minus className="size-4 shrink-0" aria-hidden />
              Remove
            </Button>
          ) : null}
          {effectiveState === "STATE_REQUESTED" && !isDirty ? (
            <Button type="button" variant="outline" size="sm" disabled>
              <Plus className="size-4 shrink-0" aria-hidden />
              Requested
            </Button>
          ) : null}
        </span>
      </PermissionTooltip>
    </td>
  );
}

type CapabilityRowProps = {
  capability: DisplayCapability;
  capabilityStates: Record<string, IntegrationCapabilityStateState>;
  actionDisabled: boolean;
  canUpdateIntegrations: boolean;
  permissionsLoading: boolean;
  onQueueCapabilityStateChange: (capability: DisplayCapability, nextState: IntegrationCapabilityStateState) => void;
};

function CapabilityRow({
  capability,
  capabilityStates,
  actionDisabled,
  canUpdateIntegrations,
  permissionsLoading,
  onQueueCapabilityStateChange,
}: CapabilityRowProps) {
  const serverState = capability.state || DEFAULT_CAPABILITY_STATE;
  const effectiveState = capabilityStates[capability.name] ?? serverState;
  const isDirty = effectiveState !== serverState;
  const description = getCapabilityDescription(capability);

  return (
    <tr className={cn(isDirty && "bg-amber-50 dark:bg-amber-950/25")}>
      <CapabilityStatusCell capability={capability} effectiveState={effectiveState} description={description} />
      <CapabilityActionsCell
        capability={capability}
        effectiveState={effectiveState}
        serverState={serverState}
        isDirty={isDirty}
        actionDisabled={actionDisabled}
        canUpdateIntegrations={canUpdateIntegrations}
        permissionsLoading={permissionsLoading}
        onQueueCapabilityStateChange={onQueueCapabilityStateChange}
      />
    </tr>
  );
}

export function CapabilitySection({
  section,
  capabilityByName,
  capabilityStates,
  canUpdateIntegrations,
  permissionsLoading,
  capabilitiesMutationPending,
  onQueueCapabilityStateChange,
}: CapabilitySectionProps) {
  const actionDisabled = !canUpdateIntegrations || capabilitiesMutationPending;

  return (
    <div
      className="overflow-hidden rounded-md border border-gray-300 dark:border-gray-600"
      role={section.label ? "group" : undefined}
      aria-label={section.label ? `${section.label} capabilities` : undefined}
    >
      {section.label ? (
        <div className="border-b border-gray-200 bg-gray-50 px-4 py-2.5 text-sm font-medium text-gray-900 dark:border-gray-800 dark:bg-gray-800/50 dark:text-gray-100">
          {section.label}
        </div>
      ) : null}
      <div className={cn(section.label && "-mt-px", "overflow-x-auto")}>
        <table className="table-fixed w-full min-w-[520px] divide-y divide-gray-200 dark:divide-gray-800">
          <colgroup>
            <col className="w-[77%]" />
            <col className="w-[23%]" />
          </colgroup>
          <tbody className="divide-y divide-gray-200 bg-white dark:divide-gray-800 dark:bg-gray-900">
            {section.names.map((capabilityName, rowIndex) => {
              const capability = capabilityByName.get(capabilityName);
              if (!capability) {
                return null;
              }
              return (
                <CapabilityRow
                  key={`${section.key}-${capability.name || rowIndex}`}
                  capability={capability}
                  capabilityStates={capabilityStates}
                  actionDisabled={actionDisabled}
                  canUpdateIntegrations={canUpdateIntegrations}
                  permissionsLoading={permissionsLoading}
                  onQueueCapabilityStateChange={onQueueCapabilityStateChange}
                />
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
