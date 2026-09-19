import { useEffect, useMemo, useState } from "react";
import { Editor } from "@monaco-editor/react";
import { Copy, Download } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useTheme } from "@/contexts/useTheme";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { CANVAS_YAML_PATH, CONSOLE_YAML_PATH } from "@/pages/app/lib/workflow-spec-paths";

export type CanvasSpecYamlTab = typeof CANVAS_YAML_PATH | typeof CONSOLE_YAML_PATH;

export type CanvasSpecYamlModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  canvasYaml: string;
  consoleYaml: string;
};

export function CanvasSpecYamlModal({ open, onOpenChange, canvasYaml, consoleYaml }: CanvasSpecYamlModalProps) {
  const [activeTab, setActiveTab] = useState<CanvasSpecYamlTab>(CANVAS_YAML_PATH);

  useEffect(() => {
    if (open) {
      setActiveTab(CANVAS_YAML_PATH);
    }
  }, [open]);

  const yamlByTab = useMemo(
    () => ({
      [CANVAS_YAML_PATH]: canvasYaml,
      [CONSOLE_YAML_PATH]: consoleYaml,
    }),
    [canvasYaml, consoleYaml],
  );
  const activeYaml = yamlByTab[activeTab];

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        size="large"
        className="flex h-full max-h-[90vh] w-[90vw] flex-col gap-0 overflow-hidden p-0 dark:border-gray-600 dark:bg-gray-900"
        data-testid="canvas-spec-yaml-modal"
      >
        <DialogHeader className="border-b border-slate-200 px-4 py-3 dark:border-gray-600">
          <DialogTitle>View YAML</DialogTitle>
          <DialogDescription>Read-only YAML for this canvas and console.</DialogDescription>
        </DialogHeader>

        <Tabs
          value={activeTab}
          onValueChange={(value) => setActiveTab(value as CanvasSpecYamlTab)}
          className="flex min-h-0 flex-1 flex-col"
        >
          <div className="flex items-center justify-between border-b border-slate-200 bg-white px-4 py-2 dark:border-gray-600 dark:bg-gray-900">
            <TabsList>
              <TabsTrigger value={CANVAS_YAML_PATH} data-testid="canvas-spec-yaml-tab-canvas">
                {CANVAS_YAML_PATH}
              </TabsTrigger>
              <TabsTrigger value={CONSOLE_YAML_PATH} data-testid="canvas-spec-yaml-tab-console">
                {CONSOLE_YAML_PATH}
              </TabsTrigger>
            </TabsList>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => void copyYaml(activeYaml)}
                data-testid="canvas-spec-yaml-copy"
              >
                <Copy className="mr-1 h-3.5 w-3.5" />
                Copy
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => downloadYaml(activeYaml, activeTab)}
                data-testid="canvas-spec-yaml-download"
              >
                <Download className="mr-1 h-3.5 w-3.5" />
                Download
              </Button>
            </div>
          </div>

          <div className="min-h-0 flex-1">
            <SpecYamlEditor value={activeYaml} />
          </div>
        </Tabs>

        <DialogFooter className="border-t border-slate-200 px-4 py-3 dark:border-gray-600">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function SpecYamlEditor({ value }: { value: string }) {
  const { resolvedTheme } = useTheme();
  const monacoTheme = resolvedTheme === "dark" ? "vs-dark" : "vs";

  return (
    <div className="h-full min-h-0" data-testid="canvas-spec-yaml-editor">
      <Editor
        height="100%"
        language="yaml"
        value={value}
        theme={monacoTheme}
        options={{
          readOnly: true,
          domReadOnly: true,
          minimap: { enabled: false },
          fontSize: 13,
          lineNumbers: "on",
          wordWrap: "on",
          folding: true,
          scrollBeyondLastLine: false,
          renderWhitespace: "boundary",
          smoothScrolling: true,
          tabSize: 2,
          renderLineHighlight: "line",
        }}
      />
    </div>
  );
}

async function copyYaml(text: string) {
  try {
    await navigator.clipboard.writeText(text);
    showSuccessToast("YAML copied to clipboard");
  } catch {
    showErrorToast("SuperPlane could not copy the YAML.");
  }
}

function downloadYaml(text: string, filename: string) {
  const blob = new Blob([text], { type: "text/yaml;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
  showSuccessToast("YAML downloaded");
}
