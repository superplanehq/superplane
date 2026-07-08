import { useEffect, useMemo, useState, type ReactNode } from "react";
import { AlertTriangle } from "lucide-react";
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
import { useTheme } from "@/contexts/useTheme";

import { ScorecardPanelForm } from "./ScorecardPanelForm";
import { TypedPanelShell } from "./TypedPanelShell";
import { WidgetScorecard } from "./widget/WidgetScorecard";
import {
  scorecardContentToYaml,
  scorecardPropsFromContent,
  validateScorecardContent,
  type ScorecardPanelContent,
} from "./scorecardContent";

export interface ScorecardPanelEditorProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialContent: ScorecardPanelContent;
  onSave: (next: ScorecardPanelContent) => void;
  /** Sample row series (oldest → newest) the live preview reduces (mockup stand-in for `rows.map(seriesField)`). */
  sampleSeries: number[];
}

type EditorTab = "form" | "yaml";

/**
 * Ground-up edit experience for the scorecard panel, presented as a
 * self-contained replica of the real `PanelEditorDialog` chrome plus an
 * always-on live preview. Staged as a prototype: it is not wired into
 * `panelTypes.ts` or the panel router.
 */
export function ScorecardPanelEditor({
  open,
  onOpenChange,
  initialContent,
  onSave,
  sampleSeries,
}: ScorecardPanelEditorProps) {
  const [tab, setTab] = useState<EditorTab>("form");
  const [draft, setDraft] = useState<ScorecardPanelContent>(initialContent);

  useEffect(() => {
    if (open) {
      setDraft(initialContent);
      setTab("form");
    }
  }, [open, initialContent]);

  const error = validateScorecardContent(draft);

  const handleSave = () => {
    if (error) return;
    onSave(draft);
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="gap-0 overflow-hidden sm:max-w-3xl dark:border-gray-600 dark:bg-gray-900"
        closeButtonClassName="top-2 right-2"
      >
        <div className="flex w-full flex-col">
          <Tabs value={tab} onValueChange={(value) => setTab(value as EditorTab)} className="flex w-full flex-col">
            <div className="-mx-6 -mt-6 border-b border-slate-950/10 bg-background px-6 pb-3 pt-5 dark:border-gray-600">
              <DialogHeader className="text-center sm:text-left">
                <DialogTitle className="mb-1 text-base font-medium">Edit Scorecard panel</DialogTitle>
                <DialogDescription className="text-gray-800 dark:text-gray-400">
                  A single KPI scored against a goal, with an optional trend, sparkline, and progress bar.
                </DialogDescription>
              </DialogHeader>
              <TabsList className="mt-3 dark:bg-gray-800">
                <TabsTrigger
                  value="form"
                  data-testid="scorecard-editor-tab-form"
                  className="dark:data-[state=active]:border-gray-600 dark:data-[state=active]:bg-gray-900"
                >
                  Form
                </TabsTrigger>
                <TabsTrigger
                  value="yaml"
                  data-testid="scorecard-editor-tab-yaml"
                  className="dark:data-[state=active]:border-gray-600 dark:data-[state=active]:bg-gray-900"
                >
                  YAML
                </TabsTrigger>
              </TabsList>
            </div>

            <TabsContent value="form" className="-mx-6 max-h-[60vh] overflow-y-auto px-6 pb-12 pt-6">
              <div className="flex flex-col gap-6">
                <LivePreview draft={draft} sampleSeries={sampleSeries} />
                <ScorecardPanelForm value={draft} onChange={setDraft} />
              </div>
            </TabsContent>

            <TabsContent value="yaml" className="mt-3">
              <YamlPreview draft={draft} />
            </TabsContent>
          </Tabs>

          <div className="-mx-6 -mb-6 flex flex-col gap-2 border-t border-slate-950/10 bg-background px-6 py-4 dark:border-gray-600">
            <EditorError message={error} />
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="button" onClick={handleSave} disabled={Boolean(error)} data-testid="scorecard-editor-save">
                Save
              </Button>
            </DialogFooter>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function LivePreview({ draft, sampleSeries }: { draft: ScorecardPanelContent; sampleSeries: number[] }) {
  const props = useMemo(() => scorecardPropsFromContent(draft, sampleSeries), [draft, sampleSeries]);
  return (
    <div className="flex flex-col gap-2">
      <span className="text-[11px] font-medium uppercase tracking-wide text-slate-400 dark:text-gray-500">Preview</span>
      <div className="rounded-lg bg-slate-100 p-4 dark:bg-gray-800/50">
        <div className="mx-auto h-[200px] w-[340px]">
          <TypedPanelShell
            title={draft.title || draft.label}
            fallbackTitle="Scorecard"
            readOnly
            onEdit={() => {}}
            onDelete={() => {}}
          >
            <WidgetScorecard {...props} />
          </TypedPanelShell>
        </div>
      </div>
    </div>
  );
}

function YamlPreview({ draft }: { draft: ScorecardPanelContent }) {
  const { resolvedTheme } = useTheme();
  const monacoTheme = resolvedTheme === "dark" ? "vs-dark" : "vs";
  const yamlText = useMemo(() => scorecardContentToYaml(draft), [draft]);
  return (
    <div className="overflow-hidden rounded-md border border-slate-200 dark:border-gray-600">
      <Editor
        height="50vh"
        language="yaml"
        value={yamlText}
        theme={monacoTheme}
        options={{
          readOnly: true,
          minimap: { enabled: false },
          fontSize: 12,
          scrollBeyondLastLine: false,
          tabSize: 2,
          automaticLayout: true,
        }}
      />
    </div>
  );
}

function EditorError({ message }: { message: string | null }): ReactNode {
  if (!message) return null;
  return (
    <div
      className="flex items-start gap-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-200"
      data-testid="scorecard-editor-error"
    >
      <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
      <span>{message}</span>
    </div>
  );
}
