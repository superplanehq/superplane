import { load } from "js-yaml";
import { useCustomAutomationSourceDraft } from "@/hooks/useCustomAutomationSourceDraft";
import { CustomAutomationSourceEditor } from "./CustomAutomationSourceEditor";
import { ThemeProvider } from "@/contexts/ThemeProvider";
import { CustomAutomationSetupForm } from "./CustomAutomationSetupForm";
import { useCustomAutomationSetupPage } from "./useCustomAutomationSetupPage";
import type { CustomAutomationColumn } from "../lib/customAutomation";

export function Editor({
  columnKey,
  onClose = () => {},
}: {
  columnKey?: CustomAutomationColumn;
  onClose?: () => void;
}) {
  const model = useCustomAutomationSetupPage({
    organizationId: "org",
    factoryId: "factory",
    factoryKey: "workspace",
    automationId: "automation",
    columnKey,
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

export function SourceEditor({
  source,
  save,
  automationId,
  stage = "backlog",
}: {
  source: string;
  save: (definition: unknown) => void;
  automationId?: string;
  stage?: CustomAutomationColumn;
}) {
  const draft = useCustomAutomationSourceDraft(source, stage);
  return (
    <CustomAutomationSourceEditor
      model={draft}
      automationId={automationId}
      stage={stage}
      saving={false}
      deleting={false}
      error=""
      onSave={() => save(load(draft.source))}
      onClose={() => {}}
    />
  );
}
