import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Bot, History, Settings, Workflow } from "lucide-react";
import { SettingsAutomationHeaderRow } from "./SettingsAutomationWorkspace";
import type { BacklogSettingsTab } from "@/hooks/useBacklogRunNavigation";

export function BacklogSettingsTabs({
  tab,
  onTabChange,
  hasAgent,
  historyEnabled,
  label,
  testId,
}: {
  tab: BacklogSettingsTab;
  onTabChange: (tab: BacklogSettingsTab) => void;
  hasAgent: boolean;
  historyEnabled: boolean;
  label: string;
  testId: "intake-settings" | "planning-settings";
}) {
  const historyTab = historyEnabled ? "runs" : "automation";
  return (
    <SettingsAutomationHeaderRow
      tabs={
        <Tabs value={tab} onValueChange={(value) => onTabChange(value as BacklogSettingsTab)}>
          <TabsList aria-label={label}>
            <TabsTrigger value="general" data-testid={`${testId}-tab-general`}>
              <Settings aria-hidden />
              General
            </TabsTrigger>
            {hasAgent ? (
              <TabsTrigger value="agent" data-testid={`${testId}-tab-agent`}>
                <Bot aria-hidden />
                Agent
              </TabsTrigger>
            ) : null}
            <TabsTrigger value={historyTab} data-testid={`${testId}-tab-${historyTab}`}>
              {historyEnabled ? <History aria-hidden /> : <Workflow aria-hidden />}
              {historyEnabled ? "Runs" : "Automation"}
            </TabsTrigger>
          </TabsList>
        </Tabs>
      }
    />
  );
}
