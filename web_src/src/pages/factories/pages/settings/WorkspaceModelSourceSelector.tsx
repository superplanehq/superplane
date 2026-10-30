import { Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { useWorkspaceModelSettings } from "./useWorkspaceModelSettings";

type Settings = ReturnType<typeof useWorkspaceModelSettings>;

function sourceOptions(settings: Settings) {
  const ownKeyAvailable = settings.providers.length > 0;
  const ownKeyDisabled = !ownKeyAvailable || (!settings.canCreateIntegration && settings.connected.length === 0);
  return [
    { value: "hosted", label: "SuperPlane", visible: true, disabled: false },
    {
      value: "own-key",
      label: "Your API key",
      visible: ownKeyAvailable || settings.source === "own-key",
      disabled: ownKeyDisabled,
    },
    {
      value: "subscription",
      label: "ChatGPT subscriptions",
      visible: settings.subscription.offered || settings.source === "subscription",
      disabled: !settings.subscription.offered && settings.source !== "subscription",
    },
  ].filter((option) => option.visible);
}

export function WorkspaceModelSourceSelector({ settings }: { settings: Settings }) {
  return (
    <div role="group" aria-label="Model source" className="grid gap-3 sm:grid-cols-3">
      {sourceOptions(settings).map((option) => {
        const active = settings.source === option.value;
        return (
          <div
            key={option.value}
            className={cn(
              "flex items-center justify-between gap-3 rounded-md border border-border bg-background p-3 sm:flex-col sm:items-start",
              active && "border-foreground/40 bg-muted/40",
            )}
          >
            <span className="text-[13px] font-medium">{option.label}</span>
            {active ? (
              <span
                role="status"
                aria-label={`${option.label} is active`}
                className="flex h-7 items-center gap-1.5 text-xs"
              >
                <Check className="size-3.5" aria-hidden />
                Active
              </span>
            ) : (
              <Button
                type="button"
                variant="outline"
                size="sm"
                aria-label={`Switch to ${option.label}`}
                disabled={!settings.canUpdate || settings.switcher.saving || option.disabled}
                onClick={() => settings.chooseSource(option.value)}
              >
                Switch
              </Button>
            )}
          </div>
        );
      })}
    </div>
  );
}
