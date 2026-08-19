import { useCallback, useMemo } from "react";
import { Copy } from "lucide-react";
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
import { useTheme } from "@/contexts/useTheme";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import type { CanvasesCanvas } from "@/api-client";
import { buildCanvasYamlFromWorkflow } from "@/pages/app/lib/canvas-yaml-staging";

type FactoryCanvasYamlModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  canvas?: CanvasesCanvas | null;
};

export function FactoryCanvasYamlModal({ open, onOpenChange, canvas }: FactoryCanvasYamlModalProps) {
  const yaml = useMemo(() => (canvas ? buildCanvasYamlFromWorkflow(canvas) : ""), [canvas]);
  const { resolvedTheme } = useTheme();
  const monacoTheme = resolvedTheme === "dark" ? "vs-dark" : "vs";

  const handleCopy = useCallback(async () => {
    try {
      await navigator.clipboard.writeText(yaml);
      showSuccessToast("Canvas YAML copied to clipboard");
    } catch {
      showErrorToast("Failed to copy YAML to clipboard");
    }
  }, [yaml]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        size="large"
        className="flex h-full max-h-[90vh] w-[90vw] flex-col gap-0 overflow-hidden p-0"
        data-testid="factory-canvas-yaml-modal"
      >
        <DialogHeader className="border-b border-border px-4 py-3">
          <DialogTitle className="font-mono text-sm text-muted-foreground">canvas.yaml</DialogTitle>
          <DialogDescription className="sr-only">View the YAML implementation of this canvas.</DialogDescription>
        </DialogHeader>
        <div className="flex items-center justify-between border-b border-border px-4 py-2">
          <span className="text-xs text-muted-foreground">Read-only YAML for this canvas</span>
          <Button type="button" variant="outline" size="sm" onClick={handleCopy} data-testid="factory-canvas-yaml-copy">
            <Copy className="mr-1 h-3.5 w-3.5" />
            Copy
          </Button>
        </div>
        <div className="min-h-0 flex-1">
          <Editor
            height="100%"
            language="yaml"
            value={yaml}
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
        <DialogFooter className="border-t border-border px-4 py-3">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
