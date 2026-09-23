import { Paperclip } from "lucide-react";
import { useRef } from "react";

import { WORK_ORDER_VISUAL_FILE_ACCEPT } from "@/lib/workOrderFiles";
import { Button } from "@/components/ui/button";

import { CREATE_WORK_ORDER_REQUEST_COPY } from "./createWorkOrderRequestCopy";

export interface CreateWorkOrderRequestAttachButtonProps {
  disabled?: boolean;
  accept?: string;
  onAttach: (files: FileList | File[]) => void;
}

export function CreateWorkOrderRequestAttachButton({
  disabled = false,
  accept = WORK_ORDER_VISUAL_FILE_ACCEPT,
  onAttach,
}: CreateWorkOrderRequestAttachButtonProps) {
  const inputRef = useRef<HTMLInputElement>(null);

  return (
    <>
      <input
        ref={inputRef}
        type="file"
        hidden
        multiple
        accept={accept}
        data-testid="create-work-order-request-image-input"
        onChange={(event) => {
          const files = event.target.files;
          if (files && files.length > 0) {
            onAttach(files);
          }
          event.target.value = "";
        }}
      />
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="size-8 rounded-full text-muted-foreground"
        disabled={disabled}
        aria-label={CREATE_WORK_ORDER_REQUEST_COPY.attach}
        data-testid="create-work-order-request-attach"
        onClick={() => {
          if (disabled) {
            return;
          }
          inputRef.current?.click();
        }}
      >
        <Paperclip className="size-4" aria-hidden />
      </Button>
    </>
  );
}
