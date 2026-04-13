import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { MermaidDiagram } from "@/components/MermaidDiagram";

export type MermaidDiagramDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  definition: string;
  title?: string;
};

export function MermaidDiagramDialog({ open, onOpenChange, definition, title }: MermaidDiagramDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="large" className="w-[min(80vw,900px)] max-h-[85vh] flex flex-col">
        <DialogHeader>
          <DialogTitle>{title || "Proposed Canvas Flow"}</DialogTitle>
        </DialogHeader>
        <div className="flex-1 overflow-auto flex items-center justify-center px-6 pb-4">
          <MermaidDiagram definition={definition} className="w-full [&_svg]:w-full [&_svg]:h-auto [&_svg]:mx-auto" />
        </div>
      </DialogContent>
    </Dialog>
  );
}
