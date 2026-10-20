import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Bot, History, Settings, Workflow } from "lucide-react";
import { SettingsAutomationHeaderRow } from "./SettingsAutomationWorkspace";
import type { FactoryAutomationSettingsTab } from "@/hooks/useFactoryAutomationNavigation";

export function FactoryAutomationSettingsTabs({
  tab,
  onTabChange,
  hasAgent,
  hasGeneral = true,
  historyEnabled,
  label,
  testId,
}: {
  tab: FactoryAutomationSettingsTab;
  onTabChange: (tab: FactoryAutomationSettingsTab) => void;
  hasAgent: boolean;
  hasGeneral?: boolean;
  historyEnabled: boolean;
  label: string;
  testId: "intake-settings" | "planning-settings" | "pr-feedback-settings" | "column-automation-view";
}) {
  const historyTab = historyEnabled ? "runs" : "automation";
  return (
    <SettingsAutomationHeaderRow
      tabs={
        <Tabs value={tab} onValueChange={(value) => onTabChange(value as FactoryAutomationSettingsTab)}>
          <TabsList aria-label={label}>
            {hasGeneral ? (
              <TabsTrigger value="general" data-testid={`${testId}-tab-general`}>
                <Settings aria-hidden />
                General
              </TabsTrigger>
            ) : null}
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
