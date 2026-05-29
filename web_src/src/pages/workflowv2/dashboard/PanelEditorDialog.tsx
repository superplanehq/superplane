import { lazy, Suspense, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { AlertTriangle, GitCompareArrows } from "lucide-react";
import * as yaml from "js-yaml";
import { Editor } from "@monaco-editor/react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

import { PANEL_TYPE_META, validatePanelContent, type PanelType } from "./panelTypes";

const CanvasYamlDiffModal = lazy(() =>
  import("../CanvasYamlDiffModal").then((module) => ({ default: module.CanvasYamlDiffModal })),
);

export interface PanelEditorDialogProps<T extends object> {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Stable id of the panel being edited. Used as a dialog key. */
  panelId: string;
  /** Panel kind — drives the dialog title and validator. */
  panelType: PanelType;
  /** Initial content snapshot. The dialog stages a draft and only commits on Save. */
  initialContent: T;
  /** Renders the per-type form. Receives the staged form draft + setter. */
  renderForm: (props: { value: T; onChange: (next: T) => void; error: string | null }) => ReactNode;
  /** Persist the validated draft. */
  onSave: (next: T) => void;
}

type EditorTab = "form" | "yaml";

/**
 * Modal editor for typed panels (node / table / chart / number) with two
 * synchronized views:
 *  - "Form" tab — per-type structured editor (supplied via `renderForm`).
 *  - "YAML" tab — Monaco editor over the same `panel.content`.
 *
 * Switching tabs converts the staged draft. Save runs the shared
 * {@link validatePanelContent} validator before invoking `onSave`.
 */
export function PanelEditorDialog<T extends object>({
  open,
  onOpenChange,
  panelId,
  panelType,
  initialContent,
  renderForm,
  onSave,
}: PanelEditorDialogProps<T>) {
  const initialYaml = useMemo(() => contentToYaml(initialContent), [initialContent]);
  const [tab, setTab] = useState<EditorTab>("form");
  const [formDraft, setFormDraft] = useState<T>(initialContent);
  const [yamlDraft, setYamlDraft] = useState<string>(initialYaml);
  const [yamlSyntaxError, setYamlSyntaxError] = useState<string | null>(null);
  const [diffOpen, setDiffOpen] = useState(false);
  const lastSyncedFromRef = useRef<EditorTab>("form");

  // Reset draft state any time the dialog re-opens for a different panel.
  useEffect(() => {
    if (open) {
      setFormDraft(initialContent);
      setYamlDraft(initialYaml);
      setYamlSyntaxError(null);
      setDiffOpen(false);
      setTab("form");
      lastSyncedFromRef.current = "form";
    }
  }, [open, initialContent, initialYaml, panelId]);

  // Keep the inactive tab in sync with edits from the active one. This way
  // toggling between Form and YAML never loses changes the user just made.
  const handleFormChange = (next: T) => {
    setFormDraft(next);
    setYamlDraft(contentToYaml(next));
    setYamlSyntaxError(null);
    lastSyncedFromRef.current = "form";
  };
  const handleYamlChange = (next: string | undefined) => {
    const value = next ?? "";
    setYamlDraft(value);
    const parsed = parseYamlObject(value);
    if (parsed.ok) {
      setFormDraft(parsed.value as T);
      setYamlSyntaxError(null);
    } else {
      setYamlSyntaxError(parsed.error);
    }
    lastSyncedFromRef.current = "yaml";
  };

  const draftForValidation: T = useMemo(() => {
    // If the YAML tab currently has a syntax error, keep validating the last
    // good form value so the user can switch back and fix it without the
    // validator screaming about missing fields.
    if (lastSyncedFromRef.current === "yaml" && yamlSyntaxError) return formDraft;
    return formDraft;
  }, [formDraft, yamlSyntaxError]);

  const schemaError = validatePanelContent(panelType, draftForValidation);
  const blockingError = yamlSyntaxError ?? schemaError;
  const hasYamlChanges = yamlDraft !== initialYaml;

  const handleSave = () => {
    if (blockingError) return;
    onSave(formDraft);
    onOpenChange(false);
  };

  useEffect(() => {
    if ((!open || !hasYamlChanges) && diffOpen) {
      setDiffOpen(false);
    }
  }, [diffOpen, hasYamlChanges, open]);

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="sm:max-w-3xl">
          <PanelEditorHeader
            panelType={panelType}
            hasYamlChanges={hasYamlChanges}
            onShowDiff={() => setDiffOpen(true)}
          />
          <PanelEditorTabs
            tab={tab}
            onTabChange={setTab}
            formContent={renderForm({ value: formDraft, onChange: handleFormChange, error: schemaError })}
            yamlDraft={yamlDraft}
            onYamlChange={handleYamlChange}
          />
          <PanelEditorError message={blockingError} />
          <PanelEditorFooter
            hasBlockingError={Boolean(blockingError)}
            onCancel={() => onOpenChange(false)}
            onSave={handleSave}
          />
        </DialogContent>
      </Dialog>
      <PanelYamlDiffModal
        open={open && diffOpen && hasYamlChanges}
        onOpenChange={setDiffOpen}
        initialYaml={initialYaml}
        draftYaml={yamlDraft}
        filename={`${panelId}.yaml`}
      />
    </>
  );
}

function PanelEditorTabs({
  tab,
  onTabChange,
  formContent,
  yamlDraft,
  onYamlChange,
}: {
  tab: EditorTab;
  onTabChange: (tab: EditorTab) => void;
  formContent: ReactNode;
  yamlDraft: string;
  onYamlChange: (value: string | undefined) => void;
}) {
  return (
    <Tabs value={tab} onValueChange={(value) => onTabChange(value as EditorTab)} className="w-full">
      <TabsList>
        <TabsTrigger value="form" data-testid="panel-editor-tab-form">
          Form
        </TabsTrigger>
        <TabsTrigger value="yaml" data-testid="panel-editor-tab-yaml">
          YAML
        </TabsTrigger>
      </TabsList>
      <TabsContent value="form" className="mt-3 max-h-[60vh] overflow-auto px-1 pb-6">
        {formContent}
      </TabsContent>
      <TabsContent value="yaml" className="mt-3">
        <div className="overflow-hidden rounded-md border border-slate-200">
          <Editor
            height="50vh"
            language="yaml"
            value={yamlDraft}
            onChange={onYamlChange}
            options={{
              minimap: { enabled: false },
              fontSize: 12,
              scrollBeyondLastLine: false,
              tabSize: 2,
              automaticLayout: true,
            }}
          />
        </div>
      </TabsContent>
    </Tabs>
  );
}

function PanelEditorHeader({
  panelType,
  hasYamlChanges,
  onShowDiff,
}: {
  panelType: PanelType;
  hasYamlChanges: boolean;
  onShowDiff: () => void;
}) {
  return (
    <DialogHeader>
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <DialogTitle>Edit {PANEL_TYPE_META[panelType].label} panel</DialogTitle>
          <DialogDescription>{PANEL_TYPE_META[panelType].description}</DialogDescription>
        </div>
        {hasYamlChanges ? (
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="shrink-0"
            onClick={onShowDiff}
            data-testid="panel-editor-show-diff"
          >
            <GitCompareArrows className="mr-1 h-3.5 w-3.5" />
            View diff
          </Button>
        ) : null}
      </div>
    </DialogHeader>
  );
}

function PanelEditorError({ message }: { message: string | null }) {
  if (!message) return null;

  return (
    <div
      className="mt-2 flex items-start gap-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800"
      data-testid="panel-editor-error"
    >
      <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
      <span>{message}</span>
    </div>
  );
}

function PanelEditorFooter({
  hasBlockingError,
  onCancel,
  onSave,
}: {
  hasBlockingError: boolean;
  onCancel: () => void;
  onSave: () => void;
}) {
  return (
    <DialogFooter className="mt-2">
      <Button type="button" variant="ghost" onClick={onCancel}>
        Cancel
      </Button>
      <Button type="button" onClick={onSave} disabled={hasBlockingError} data-testid="panel-editor-save">
        Save
      </Button>
    </DialogFooter>
  );
}

function PanelYamlDiffModal({
  open,
  onOpenChange,
  initialYaml,
  draftYaml,
  filename,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialYaml: string;
  draftYaml: string;
  filename: string;
}) {
  return (
    <Suspense fallback={null}>
      <CanvasYamlDiffModal
        open={open}
        onOpenChange={onOpenChange}
        liveYamlText={initialYaml}
        draftYamlText={draftYaml}
        filename={filename}
        title="Panel YAML diff"
        dialogTitle="Panel YAML diff"
        description="Side-by-side YAML comparison between the saved panel content and the current panel edits."
        liveLabel="Saved"
        draftLabel="Draft edits"
      />
    </Suspense>
  );
}

function contentToYaml(content: object): string {
  return yaml.dump(content ?? {}, { noRefs: true, lineWidth: 100, sortKeys: false });
}

function parseYamlObject(text: string): { ok: true; value: Record<string, unknown> } | { ok: false; error: string } {
  try {
    const parsed = yaml.load(text);
    if (parsed == null) return { ok: true, value: {} };
    if (typeof parsed !== "object" || Array.isArray(parsed)) {
      return { ok: false, error: "YAML must be an object at the root." };
    }
    return { ok: true, value: parsed as Record<string, unknown> };
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : "Invalid YAML." };
  }
}
