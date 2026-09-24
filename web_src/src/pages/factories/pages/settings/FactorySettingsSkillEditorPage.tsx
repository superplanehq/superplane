import { Editor } from "@monaco-editor/react";

import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { useTheme } from "@/contexts/useTheme";
import { usePageTitle } from "@/hooks/usePageTitle";

import { FactoryDeleteDialog } from "../../FactoryDeleteDialog";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { FactorySettingsPageFrame } from "./FactorySettingsCard";
import { useSkillEditorPage } from "./useSkillEditorPage";

export function FactorySettingsSkillEditorPage() {
  const editor = useSkillEditorPage();
  const { resolvedTheme } = useTheme();
  usePageTitle([
    editor.isCreate ? AGENT_RESOURCES_COPY.addSkill : editor.name || AGENT_RESOURCES_COPY.editSkill,
    AGENT_RESOURCES_COPY.skillsTitle,
    editor.factory.name ?? "Workspace",
  ]);

  if (!editor.isCreate && !editor.skillsLoading && !editor.resource) {
    return (
      <FactorySettingsPageFrame title={AGENT_RESOURCES_COPY.editSkill} wide>
        <p className="text-[13px] text-destructive">{AGENT_RESOURCES_COPY.skillNotFound}</p>
      </FactorySettingsPageFrame>
    );
  }

  const displayName = editor.name.trim() || AGENT_RESOURCES_COPY.unnamedSkill;

  return (
    <FactorySettingsPageFrame
      title={editor.isCreate ? AGENT_RESOURCES_COPY.addSkill : AGENT_RESOURCES_COPY.editSkill}
      subtitle={AGENT_RESOURCES_COPY.skillDialogDescription}
      wide
      actions={<SkillEditorActions editor={editor} />}
    >
      <SkillEditorFields
        name={editor.name}
        markdown={editor.markdown}
        command={editor.command}
        commandName={editor.commandName}
        nameError={editor.nameError}
        markdownError={editor.markdownError}
        resolvedTheme={resolvedTheme}
        onNameChange={editor.setName}
        onMarkdownChange={editor.setMarkdown}
      />
      <FactoryDeleteDialog
        open={editor.pendingDelete}
        factoryName={displayName}
        title={`Delete "${displayName}"?`}
        description={AGENT_RESOURCES_COPY.deleteSkillDescription}
        canDelete={editor.canUpdate}
        isDeleting={editor.isDeleting}
        onClose={() => editor.setPendingDelete(false)}
        onConfirm={editor.confirmDelete}
      />
    </FactorySettingsPageFrame>
  );
}

function SkillEditorActions({ editor }: { editor: ReturnType<typeof useSkillEditorPage> }) {
  return (
    <div className="flex items-center gap-2">
      {!editor.isCreate ? (
        <Button
          type="button"
          variant="outline"
          onClick={() => editor.setPendingDelete(true)}
          disabled={!editor.canUpdate}
        >
          {AGENT_RESOURCES_COPY.delete}
        </Button>
      ) : null}
      <Button type="button" variant="outline" onClick={editor.navigateToList}>
        {AGENT_RESOURCES_COPY.cancel}
      </Button>
      <PermissionTooltip allowed={editor.canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
        <LoadingButton
          type="button"
          onClick={() => void editor.handleSave()}
          loading={editor.isSaving}
          disabled={!editor.canUpdate}
          data-testid="agent-resource-skill-save"
        >
          {editor.isCreate ? AGENT_RESOURCES_COPY.addSkill : AGENT_RESOURCES_COPY.saveSkill}
        </LoadingButton>
      </PermissionTooltip>
    </div>
  );
}

function SkillEditorFields({
  name,
  markdown,
  command,
  commandName,
  nameError,
  markdownError,
  resolvedTheme,
  onNameChange,
  onMarkdownChange,
}: {
  name: string;
  markdown: string;
  command: string;
  commandName: string;
  nameError: string;
  markdownError: string;
  resolvedTheme: string;
  onNameChange: (value: string) => void;
  onMarkdownChange: (value: string) => void;
}) {
  return (
    <div className="flex min-h-[70vh] flex-col gap-4" data-testid="factory-settings-skill-editor">
      <SkillNameFields
        name={name}
        command={command}
        commandName={commandName}
        nameError={nameError}
        onNameChange={onNameChange}
      />
      <div className="flex min-h-0 flex-1 flex-col gap-1.5">
        <Label>{AGENT_RESOURCES_COPY.markdownLabel}</Label>
        <div
          className="min-h-[480px] flex-1 overflow-hidden rounded-lg border border-border"
          data-testid="agent-resource-skill-markdown"
        >
          <Editor
            height="100%"
            defaultLanguage="markdown"
            language="markdown"
            value={markdown}
            theme={resolvedTheme === "dark" ? "vs-dark" : "vs"}
            onChange={(next) => onMarkdownChange(next ?? "")}
            options={{
              minimap: { enabled: false },
              wordWrap: "on",
              fontSize: 13,
              scrollBeyondLastLine: false,
              automaticLayout: true,
            }}
          />
        </div>
        <p className="text-[12px] text-muted-foreground">{AGENT_RESOURCES_COPY.markdownHelper}</p>
        {markdownError ? <p className="text-[12px] text-destructive">{markdownError}</p> : null}
      </div>
    </div>
  );
}

function SkillNameFields({
  name,
  command,
  commandName,
  nameError,
  onNameChange,
}: {
  name: string;
  command: string;
  commandName: string;
  nameError: string;
  onNameChange: (value: string) => void;
}) {
  return (
    <div className="grid gap-4 md:grid-cols-2">
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="skill-editor-name">{AGENT_RESOURCES_COPY.nameLabel}</Label>
        <Input
          id="skill-editor-name"
          data-testid="agent-resource-skill-name"
          value={name}
          onChange={(event) => onNameChange(event.target.value)}
          placeholder="Review copy"
          autoComplete="off"
        />
        <p className="text-[12px] text-muted-foreground">{AGENT_RESOURCES_COPY.skillNameHelper}</p>
        {nameError ? <p className="text-[12px] text-destructive">{nameError}</p> : null}
      </div>
      <div className="flex flex-col gap-1.5">
        <Label htmlFor="skill-editor-command">{AGENT_RESOURCES_COPY.skillCommandLabel}</Label>
        <Input id="skill-editor-command" data-testid="agent-resource-skill-command" value={command} readOnly />
        <p className="text-[12px] text-muted-foreground">
          {commandName
            ? AGENT_RESOURCES_COPY.skillCommandHelper(commandName)
            : AGENT_RESOURCES_COPY.skillCommandEmptyHelper}
        </p>
      </div>
    </div>
  );
}
