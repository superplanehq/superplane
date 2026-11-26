import { ThemeProvider } from "@/contexts/ThemeProvider";
import { CustomAutomationSetupForm } from "./CustomAutomationSetupForm";
import { useCustomAutomationSetupPage } from "./useCustomAutomationSetupPage";

export function Editor({
  columnKey,
  onClose = () => {},
}: {
  columnKey?: "backlog" | "verify" | `custom-${string}`;
  onClose?: () => void;
}) {
  const model = useCustomAutomationSetupPage({
    organizationId: "org",
    factoryId: "factory",
    factoryKey: "workspace",
    automationId: "automation",
  });
  return (
    <ThemeProvider>
      <CustomAutomationSetupForm
        name={model.name}
        icon={model.icon}
        triggers={model.triggers}
        runsOnTask={model.runsOnTask}
        columnKey={columnKey}
        draft={model.draft}
        organizationId="org"
        factoryId="factory"
        factoryKey="workspace"
        saving={model.saving}
        deleting={false}
        error={model.saveError}
        onNameChange={model.setName}
        onIconChange={model.setIcon}
        onTriggersChange={model.setTriggers}
        onRunsOnTaskChange={(runsOnTask) => model.setRunsOnTask(runsOnTask ?? true)}
        onDraftChange={model.setDraft}
        onSave={() => void model.save()}
        onClose={onClose}
      />
    </ThemeProvider>
  );
}
