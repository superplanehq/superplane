import { TabsList, TabsTrigger } from "@/components/ui/tabs";
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
    <TabsList
      aria-label="Model source"
      className="h-auto w-full flex-col items-stretch rounded-xl sm:w-fit sm:flex-row sm:rounded-full"
    >
      {sourceOptions(settings).map((option) => (
        <TabsTrigger
          key={option.value}
          value={option.value}
          disabled={!settings.canUpdate || settings.switcher.saving || option.disabled}
          className="justify-start rounded-lg sm:justify-center sm:rounded-full"
          onKeyDown={(event) => {
            if (event.key !== "Enter" && event.key !== " ") return;
            event.preventDefault();
            settings.chooseSource(option.value);
          }}
        >
          {option.label}
        </TabsTrigger>
      ))}
    </TabsList>
  );
}
