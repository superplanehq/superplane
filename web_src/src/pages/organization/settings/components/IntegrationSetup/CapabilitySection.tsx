import type { IntegrationsCapabilityDefinition } from "@/api-client";
import type { CapabilityGroupSection } from "@/lib/capabilities";
import { cn } from "@/lib/utils";
import { CopyButton } from "@/ui/CopyButton";
import { Check, CircleOff, Minus } from "lucide-react";
import { getGroupToggleState } from "./lib";

function getCapabilitySelectionDotClass(selected: boolean) {
  return selected ? "bg-green-500" : "bg-content-muted";
}

export interface CapabilitySectionProps {
  section: CapabilityGroupSection;
  capabilityByName: Map<string, IntegrationsCapabilityDefinition>;
  selectedCapabilities: ReadonlySet<string>;
  onToggleCapability: (capabilityName: string) => void;
  onToggleCapabilityGroup: (capabilityNames: string[]) => void;
  selectionDisabled: boolean;
}

export function CapabilitySection({
  section,
  capabilityByName,
  selectedCapabilities,
  onToggleCapability,
  onToggleCapabilityGroup,
  selectionDisabled,
}: CapabilitySectionProps) {
  const groupState = section.label ? getGroupToggleState(section.names, selectedCapabilities) : undefined;
  const selectedInSection = section.names.filter((name) => selectedCapabilities.has(name)).length;
  const groupIcon =
    groupState === undefined ? null : groupState === "all" ? (
      <Check className="size-4 shrink-0 text-green-600 dark:text-green-400" aria-hidden />
    ) : groupState === "some" ? (
      <Minus className="size-4 shrink-0 text-amber-600 dark:text-amber-400" aria-hidden />
    ) : (
      <CircleOff className="size-4 shrink-0 text-content-muted" aria-hidden />
    );

  return (
    <div
      className="overflow-hidden rounded-md border border-edge-default"
      role={section.label ? "group" : undefined}
      aria-label={section.label ? `${section.label} capabilities` : undefined}
    >
      {section.label ? (
        <button
          type="button"
          disabled={selectionDisabled}
          aria-label={
            groupState === "all"
              ? `Remove all selections from ${section.label}`
              : `Select all capabilities in ${section.label}`
          }
          className={cn(
            "flex w-full cursor-pointer flex-wrap items-center justify-between gap-3 border-b border-edge-subtle bg-surface-subtle px-4 py-3 text-left transition-colors hover:bg-action-neutral-hover",
            selectionDisabled && "!cursor-not-allowed opacity-70 hover:bg-action-neutral-hover/50",
          )}
          onClick={() => onToggleCapabilityGroup(section.names)}
          onKeyDown={(event) => {
            if (selectionDisabled) {
              return;
            }
            if (event.key === "Enter" || event.key === " ") {
              event.preventDefault();
              onToggleCapabilityGroup(section.names);
            }
          }}
        >
          <div className="min-w-0">
            <span className="text-sm font-medium text-content-primary">{section.label}</span>
            <span className="ml-2 text-xs tabular-nums text-content-secondary">
              {selectedInSection}/{section.names.length}
            </span>
          </div>
          <div className="flex shrink-0 items-center">{groupIcon}</div>
        </button>
      ) : null}
      <div className={cn(section.label && "-mt-px", "overflow-x-auto")}>
        <table className="w-full min-w-[520px] divide-y divide-edge-subtle">
          <tbody className="divide-y divide-edge-subtle bg-surface-raised">
            {section.names.map((capabilityName) => {
              const capability = capabilityByName.get(capabilityName);
              if (!capability) {
                return null;
              }

              const checked = selectedCapabilities.has(capabilityName);
              const statusDotClass = getCapabilitySelectionDotClass(checked);

              return (
                <tr
                  key={capabilityName}
                  className={cn(
                    "transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
                    selectionDisabled
                      ? "cursor-not-allowed opacity-70"
                      : "cursor-pointer hover:bg-action-neutral-hover/60",
                  )}
                  onClick={() => onToggleCapability(capabilityName)}
                  onKeyDown={(event) => {
                    if (selectionDisabled) {
                      return;
                    }
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      onToggleCapability(capabilityName);
                    }
                  }}
                  tabIndex={selectionDisabled ? -1 : 0}
                  aria-selected={checked}
                  aria-label={`${checked ? "Selected" : "Not selected"}: ${capabilityName}. Press Enter or Space to toggle.`}
                >
                  <td className="px-4 py-3 align-middle">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className={cn("h-2.5 w-2.5 shrink-0 rounded-full", statusDotClass)} aria-hidden />
                      <span className="font-mono text-sm text-content-primary">{capabilityName}</span>
                      <CopyButton text={capabilityName} />
                    </div>
                  </td>
                  <td className="px-4 py-3 align-middle">
                    {capability.description ? (
                      <div className="text-sm text-content-secondary">{capability.description}</div>
                    ) : null}
                  </td>
                  <td className="px-4 py-3 align-middle">
                    <div className="flex justify-end">
                      {checked ? (
                        <Check className="size-3 shrink-0 text-green-600 dark:text-green-400" aria-hidden />
                      ) : (
                        <CircleOff className="size-3 shrink-0 text-content-muted" aria-hidden />
                      )}
                    </div>
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
