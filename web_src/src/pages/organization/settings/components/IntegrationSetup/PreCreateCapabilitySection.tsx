import type { IntegrationsCapabilityDefinition } from "@/api-client";
import type { CapabilityGroupSection } from "@/lib/capabilities";
import { cn } from "@/lib/utils";
import { CopyButton } from "@/ui/CopyButton";
import { Check, CircleOff, Minus } from "lucide-react";
import { getGroupToggleState } from "./lib";

function getCapabilitySelectionDotClass(selected: boolean) {
  return selected ? "bg-green-500" : "bg-gray-400 dark:bg-gray-500";
}

export interface PreCreateCapabilitySectionProps {
  section: CapabilityGroupSection;
  capabilityByName: Map<string, IntegrationsCapabilityDefinition>;
  selectedCapabilities: ReadonlySet<string>;
  onToggleCapability: (capabilityName: string) => void;
  onToggleCapabilityGroup: (capabilityNames: string[]) => void;
  isCreatePending: boolean;
}

export function PreCreateCapabilitySection({
  section,
  capabilityByName,
  selectedCapabilities,
  onToggleCapability,
  onToggleCapabilityGroup,
  isCreatePending,
}: PreCreateCapabilitySectionProps) {
  const groupState = section.label ? getGroupToggleState(section.names, selectedCapabilities) : undefined;
  const selectedInSection = section.names.filter((name) => selectedCapabilities.has(name)).length;
  const groupIcon =
    groupState === undefined ? null : groupState === "all" ? (
      <Check className="size-4 shrink-0 text-green-600 dark:text-green-400" aria-hidden />
    ) : groupState === "some" ? (
      <Minus className="size-4 shrink-0 text-amber-600 dark:text-amber-400" aria-hidden />
    ) : (
      <CircleOff className="size-4 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden />
    );

  return (
    <div
      className="overflow-hidden rounded-md border border-gray-300 dark:border-gray-700"
      role={section.label ? "group" : undefined}
      aria-label={section.label ? `${section.label} capabilities` : undefined}
    >
      {section.label ? (
        <button
          type="button"
          disabled={isCreatePending}
          aria-label={
            groupState === "all"
              ? `Remove all selections from ${section.label}`
              : `Select all capabilities in ${section.label}`
          }
          className={cn(
            "flex w-full cursor-pointer flex-wrap items-center justify-between gap-3 border-b border-gray-200 bg-gray-50 px-4 py-3 text-left transition-colors hover:bg-gray-100 dark:border-gray-800 dark:bg-gray-800/50 dark:hover:bg-gray-800",
            isCreatePending && "!cursor-not-allowed opacity-70 hover:bg-gray-50 dark:hover:bg-gray-800/50",
          )}
          onClick={() => onToggleCapabilityGroup(section.names)}
          onKeyDown={(event) => {
            if (isCreatePending) {
              return;
            }
            if (event.key === "Enter" || event.key === " ") {
              event.preventDefault();
              onToggleCapabilityGroup(section.names);
            }
          }}
        >
          <div className="min-w-0">
            <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{section.label}</span>
            <span className="ml-2 text-xs tabular-nums text-gray-500 dark:text-gray-400">
              {selectedInSection}/{section.names.length}
            </span>
          </div>
          <div className="flex shrink-0 items-center">{groupIcon}</div>
        </button>
      ) : null}
      <div className={cn(section.label && "-mt-px", "overflow-x-auto")}>
        <table className="w-full min-w-[520px] divide-y divide-gray-200 dark:divide-gray-800">
          <tbody className="divide-y divide-gray-200 bg-white dark:divide-gray-800 dark:bg-gray-900">
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
                    "transition-colors outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background dark:focus-visible:ring-offset-gray-900",
                    isCreatePending
                      ? "cursor-not-allowed opacity-70"
                      : "cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800/60",
                  )}
                  onClick={() => onToggleCapability(capabilityName)}
                  onKeyDown={(event) => {
                    if (isCreatePending) {
                      return;
                    }
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      onToggleCapability(capabilityName);
                    }
                  }}
                  tabIndex={isCreatePending ? -1 : 0}
                  aria-selected={checked}
                  aria-label={`${checked ? "Selected" : "Not selected"}: ${capabilityName}. Press Enter or Space to toggle.`}
                >
                  <td className="px-4 py-3 align-middle">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className={cn("h-2.5 w-2.5 shrink-0 rounded-full", statusDotClass)} aria-hidden />
                      <span className="font-mono text-sm text-gray-800 dark:text-gray-100">{capabilityName}</span>
                      <CopyButton text={capabilityName} />
                    </div>
                  </td>
                  <td className="px-4 py-3 align-middle">
                    {capability.description ? (
                      <div className="text-sm text-gray-600 dark:text-gray-400">{capability.description}</div>
                    ) : null}
                  </td>
                  <td className="px-4 py-3 align-middle">
                    <div className="flex justify-end">
                      {checked ? (
                        <Check className="size-3 shrink-0 text-green-600 dark:text-green-400" aria-hidden />
                      ) : (
                        <CircleOff className="size-3 shrink-0 text-gray-400 dark:text-gray-500" aria-hidden />
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
