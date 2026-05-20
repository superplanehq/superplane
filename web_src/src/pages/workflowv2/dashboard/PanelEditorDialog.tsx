import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { AlertTriangle } from "lucide-react";
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
  const lastSyncedFromRef = useRef<EditorTab>("form");

  // Reset draft state any time the dialog re-opens for a different panel.
  useEffect(() => {
    if (open) {
      setFormDraft(initialContent);
      setYamlDraft(initialYaml);
      setYamlSyntaxError(null);
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

  const handleSave = () => {
    if (blockingError) return;
    onSave(formDraft);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-3xl">
        <DialogHeader>
          <DialogTitle>Edit {PANEL_TYPE_META[panelType].label} panel</DialogTitle>
          <DialogDescription>{PANEL_TYPE_META[panelType].description}</DialogDescription>
        </DialogHeader>
        <Tabs value={tab} onValueChange={(v) => setTab(v as EditorTab)} className="w-full">
          <TabsList>
            <TabsTrigger value="form" data-testid="panel-editor-tab-form">
              Form
            </TabsTrigger>
            <TabsTrigger value="yaml" data-testid="panel-editor-tab-yaml">
              YAML
            </TabsTrigger>
          </TabsList>
          <TabsContent value="form" className="mt-3 max-h-[60vh] overflow-auto">
            {renderForm({ value: formDraft, onChange: handleFormChange, error: schemaError })}
          </TabsContent>
          <TabsContent value="yaml" className="mt-3">
            <div className="overflow-hidden rounded-md border border-slate-200">
              <Editor
                height="50vh"
                language="yaml"
                value={yamlDraft}
                onChange={handleYamlChange}
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
        {blockingError ? (
          <div
            className="mt-2 flex items-start gap-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800"
            data-testid="panel-editor-error"
          >
            <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
            <span>{blockingError}</span>
          </div>
        ) : null}
        <DialogFooter className="mt-2">
          <Button type="button" variant="ghost" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="button" onClick={handleSave} disabled={Boolean(blockingError)} data-testid="panel-editor-save">
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
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
