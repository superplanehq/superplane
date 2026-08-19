import { useMemo } from "react";

import type { CanvasesCanvas } from "@/api-client";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { buildCanvasYamlFromWorkflow } from "@/pages/app/lib/canvas-yaml-staging";
import { CopyButton } from "@/ui/CopyButton";
import { FactoryCanvasYamlEditor } from "./FactoryCanvasYamlEditor";

type FactoryCanvasYamlModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  canvas?: CanvasesCanvas | null;
};

export function FactoryCanvasYamlModal({ open, onOpenChange, canvas }: FactoryCanvasYamlModalProps) {
  const yaml = useMemo(() => (canvas ? buildCanvasYamlFromWorkflow(canvas) : ""), [canvas]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        size="large"
        className="flex h-full max-h-[90vh] w-[90vw] flex-col gap-0 overflow-hidden p-0"
        data-testid="factory-canvas-yaml-modal"
      >
        <DialogHeader className="border-b border-border px-5 py-4">
          <DialogTitle>View YAML</DialogTitle>
          <DialogDescription>Read-only YAML for this canvas.</DialogDescription>
        </DialogHeader>
        <div className="min-h-0 flex-1">
          <FactoryCanvasYamlEditor value={yaml} readOnly />
        </div>
        <DialogFooter className="border-t border-border px-5 py-3">
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
          <CopyButton
            variant="button"
            buttonVariant="default"
            text={yaml}
            copiedLabel="Copied"
            data-testid="factory-canvas-yaml-copy"
          >
            Copy YAML
          </CopyButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
