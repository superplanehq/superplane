import type { CanvasMemoryEntry } from "@/hooks/useCanvasData";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";

import { CanvasMemoryView } from "./CanvasMemoryView";

export type CanvasMemoryModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  entries: CanvasMemoryEntry[];
  isLoading?: boolean;
  errorMessage?: string;
  onDeleteEntry?: (memoryId: string) => void;
  deletingId?: string;
};

export function CanvasMemoryModal(props: CanvasMemoryModalProps) {
  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent size="large" className="flex max-h-[90vh] w-[90vw] flex-col gap-0 overflow-hidden p-0">
        <DialogTitle className="sr-only">Canvas Memory</DialogTitle>
        <div className="min-h-0 flex-1 overflow-y-auto bg-slate-100">
          <CanvasMemoryView
            entries={props.entries}
            isLoading={props.isLoading}
            errorMessage={props.errorMessage}
            onDeleteEntry={props.onDeleteEntry}
            deletingId={props.deletingId}
          />
        </div>
      </DialogContent>
    </Dialog>
  );
}
