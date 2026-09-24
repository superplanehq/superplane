import { FileText, X } from "lucide-react";

import { Button } from "@/components/ui/button";
import type { UploadedWorkOrderFile } from "@/hooks/useWorkOrderFileUpload";

export function PendingWorkOrderFileChips({
  files,
  onRemove,
}: {
  files: UploadedWorkOrderFile[];
  onRemove: (id: string) => void;
}) {
  return (
    <div className="flex min-w-0 flex-wrap items-end gap-1.5" data-testid="create-work-order-request-file-chips">
      {files.map((file) => (
        <span
          key={file.id}
          className="flex max-w-44 items-center gap-1 rounded-md border bg-card px-1.5 py-1 text-[12px] text-foreground"
          data-testid={`create-work-order-request-file-${file.id}`}
        >
          <FileText className="size-3 shrink-0 text-muted-foreground" aria-hidden />
          <span className="truncate" title={file.filename}>
            {file.filename}
          </span>
          <Button
            type="button"
            variant="ghost"
            size="icon-xs"
            className="size-4 shrink-0 text-muted-foreground hover:text-foreground"
            aria-label={`Remove ${file.filename}`}
            data-testid={`create-work-order-request-file-remove-${file.id}`}
            onClick={() => onRemove(file.id)}
          >
            <X className="size-3" aria-hidden />
          </Button>
        </span>
      ))}
    </div>
  );
}
