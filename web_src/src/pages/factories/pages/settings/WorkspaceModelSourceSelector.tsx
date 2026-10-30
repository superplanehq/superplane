import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { useWorkspaceModelSettings } from "./useWorkspaceModelSettings";

type Settings = ReturnType<typeof useWorkspaceModelSettings>;

function sourceOptions(settings: Settings) {
  const ownKeyAvailable = settings.providers.length > 0;
  const ownKeyDisabled = !ownKeyAvailable || (!settings.canCreateIntegration && settings.connected.length === 0);
  return [
    {
      value: "hosted",
      label: "SuperPlane",
      description: "Use models managed by SuperPlane. SuperPlane bills these runs.",
      visible: true,
      disabled: false,
    },
    {
      value: "own-key",
      label: "Your API key",
      description: "Connect a provider key. Your provider bills these runs.",
      visible: ownKeyAvailable || settings.source === "own-key",
      disabled: ownKeyDisabled,
    },
    {
      value: "subscription",
      label: "ChatGPT subscriptions",
      description: "Use each task owner’s personal ChatGPT subscription.",
      visible: settings.subscription.offered || settings.source === "subscription",
      disabled: !settings.subscription.offered && settings.source !== "subscription",
    },
  ].filter((option) => option.visible);
}

export function WorkspaceModelSourceSelector({ settings }: { settings: Settings }) {
  const options = sourceOptions(settings);
  const canChangeProvider = settings.providers.some(
    (provider) =>
      provider !== settings.currentProvider && (settings.canCreateIntegration || settings.connected.includes(provider)),
  );
  return (
    <div
      role="group"
      aria-label="Model source"
      className={cn("grid gap-3 sm:grid-cols-2", options.length > 2 && "lg:grid-cols-3")}
    >
      {options.map((option) => {
        const active = settings.source === option.value;
        const changingProvider = active && option.value === "own-key" && canChangeProvider;
        return (
          <SourceOptionCard
            key={option.value}
            option={option}
            active={active}
            changingProvider={changingProvider}
            disabled={
              !settings.canUpdate || settings.switcher.saving || option.disabled || (active && !changingProvider)
            }
            onChoose={() => settings.chooseSource(option.value)}
          />
        );
      })}
    </div>
  );
}

function SourceOptionCard({
  option,
  active,
  changingProvider,
  disabled,
  onChoose,
}: {
  option: ReturnType<typeof sourceOptions>[number];
  active: boolean;
  changingProvider: boolean;
  disabled: boolean;
  onChoose: () => void;
}) {
  const actionLabel = changingProvider ? "Change provider" : "Current source";
  return (
    <div
      className={cn(
        "flex flex-col rounded-lg border p-4",
        active ? "border-foreground/30 bg-muted/30" : "border-border",
      )}
    >
      <div className="flex min-h-6 flex-wrap items-center justify-between gap-2">
        <h3 className="text-sm font-semibold">{option.label}</h3>
        {active ? (
          <Badge variant="outline" className="text-[10px]">
            Current
          </Badge>
        ) : null}
      </div>
      <p className="my-5 flex-1 text-xs text-muted-foreground">{option.description}</p>
      <Button
        type="button"
        variant={active ? "outline" : "default"}
        size="sm"
        aria-label={active ? undefined : `Select ${option.label}`}
        disabled={disabled}
        onClick={onChoose}
      >
        {active ? actionLabel : `Select ${option.label}`}
      </Button>
    </div>
  );
}
