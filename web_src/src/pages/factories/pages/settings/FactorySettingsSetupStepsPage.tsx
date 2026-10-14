import { Editor } from "@monaco-editor/react";

import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { useTheme } from "@/contexts/useTheme";
import { usePageTitle } from "@/hooks/usePageTitle";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { FactorySettingsPageFrame } from "./FactorySettingsCard";
import { useSetupStepsEditorPage } from "./useSetupStepsEditorPage";

export function FactorySettingsSetupStepsPage() {
  const editor = useSetupStepsEditorPage();
  const { resolvedTheme } = useTheme();
  usePageTitle([
    AGENT_RESOURCES_COPY.setupStepsTitle,
    AGENT_RESOURCES_COPY.agentTitle,
    editor.factory?.name ?? "Workspace",
  ]);

  return (
    <FactorySettingsPageFrame
      title={AGENT_RESOURCES_COPY.setupStepsTitle}
      subtitle={AGENT_RESOURCES_COPY.setupStepsHelper}
      wide
      backHref={editor.listPath}
      backLabel={AGENT_RESOURCES_COPY.backToAgent}
      backTestId="agent-setup-steps-back"
      actions={<SetupStepsEditorActions editor={editor} />}
    >
      {editor.setupLoading ? (
        <p className="text-[13px] text-muted-foreground">{AGENT_RESOURCES_COPY.setupStepsLoading}</p>
      ) : editor.setupError ? (
        <p className="text-[13px] text-destructive">{AGENT_RESOURCES_COPY.setupStepsLoadError}</p>
      ) : (
        <div
          className="h-[70vh] min-h-[480px] overflow-hidden rounded-lg border border-border"
          data-testid="agent-setup-steps-editor"
        >
          <Editor
            height="100%"
            defaultLanguage="shell"
            language="shell"
            value={editor.value}
            theme={resolvedTheme === "dark" ? "vs-dark" : "vs"}
            onChange={(next) => editor.setValue(next ?? "")}
            options={{
              minimap: { enabled: false },
              wordWrap: "on",
              fontSize: 13,
              scrollBeyondLastLine: false,
              automaticLayout: true,
            }}
          />
        </div>
      )}
    </FactorySettingsPageFrame>
  );
}

function SetupStepsEditorActions({ editor }: { editor: ReturnType<typeof useSetupStepsEditorPage> }) {
  return (
    <div className="flex items-center gap-2">
      <Button type="button" variant="outline" onClick={editor.navigateToList}>
        {AGENT_RESOURCES_COPY.cancel}
      </Button>
      <PermissionTooltip allowed={editor.canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
        <LoadingButton
          type="button"
          onClick={() => void editor.handleSave()}
          loading={editor.isSaving}
          loadingText={AGENT_RESOURCES_COPY.setupStepsSaving}
          disabled={!editor.canUpdate || !editor.isDirty || editor.isSaving || editor.setupLoading}
          data-testid="agent-setup-steps-save"
        >
          {AGENT_RESOURCES_COPY.setupStepsSave}
        </LoadingButton>
      </PermissionTooltip>
    </div>
  );
}
