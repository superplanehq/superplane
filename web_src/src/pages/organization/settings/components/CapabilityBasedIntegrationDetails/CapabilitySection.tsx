import type { IntegrationCapabilityStateState } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import type { CapabilityGroupSection } from "@/lib/capabilities";
import { cn } from "@/lib/utils";
import { CopyButton } from "@/ui/CopyButton";
import {
  DEFAULT_CAPABILITY_STATE,
  type DisplayCapability,
  getCapabilityDescription,
  getCapabilityStatusDotClass,
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
      className="overflow-hidden rounded-md border border-gray-300 dark:border-gray-700"
      role={section.label ? "group" : undefined}
      aria-label={section.label ? `${section.label} capabilities` : undefined}
    >
      {section.label ? (
        <div className="border-b border-gray-200 bg-gray-50 px-4 py-2.5 text-sm font-medium text-gray-900 dark:border-gray-800 dark:bg-gray-800/50 dark:text-gray-100">
          {section.label}
        </div>
      ) : null}
      <div className={cn(section.label && "-mt-px", "overflow-x-auto")}>
        <table className="w-full min-w-[520px] divide-y divide-gray-200 dark:divide-gray-800">
          <tbody className="divide-y divide-gray-200 bg-white dark:divide-gray-800 dark:bg-gray-900">
            {section.names.map((capabilityName, rowIndex) => {
              const capability = capabilityByName.get(capabilityName);
              if (!capability) {
                return null;
              }

              const serverState = capability.state || DEFAULT_CAPABILITY_STATE;
              const effectiveState = capabilityStates[capability.name] ?? serverState;
              const statusDotClass = getCapabilityStatusDotClass(effectiveState);
              const isDirty = effectiveState !== serverState;
              const description = getCapabilityDescription(capability);

              return (
                <tr
                  key={`${section.key}-${capability.name || rowIndex}`}
                  className={cn(isDirty && "bg-amber-50 dark:bg-amber-950/25")}
                >
                  <td className="px-4 py-3 align-middle">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className={cn("h-2.5 w-2.5 shrink-0 rounded-full", statusDotClass)} aria-hidden />
                      <span className="font-mono text-sm text-gray-800 dark:text-gray-100">{capability.name}</span>
                      <CopyButton text={capability.name} />
                    </div>
                  </td>
                  <td className="px-4 py-3 align-middle">
                    {description ? <div className="text-sm text-gray-600 dark:text-gray-400">{description}</div> : null}
                  </td>
                  <td className="px-4 py-3 align-middle text-right">
                    <PermissionTooltip
                      allowed={canUpdateIntegrations || permissionsLoading}
                      message="You don't have permission to update integrations."
                    >
                      <span className="flex justify-end">
                        {effectiveState === "STATE_ENABLED" ? (
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            disabled={actionDisabled}
                            onClick={() => onQueueCapabilityStateChange(capability, "STATE_DISABLED")}
                          >
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
                            Enable
                          </Button>
                        ) : null}
                        {effectiveState === "STATE_UNAVAILABLE" ? (
                          <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            disabled={actionDisabled}
                            onClick={() => onQueueCapabilityStateChange(capability, "STATE_REQUESTED")}
                          >
                            Request
                          </Button>
                        ) : null}
                        {effectiveState === "STATE_REQUESTED" ? (
                          <Button type="button" variant="outline" size="sm" disabled>
                            Requested
                          </Button>
                        ) : null}
                      </span>
                    </PermissionTooltip>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}
