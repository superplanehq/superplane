import { ArrowUp, Loader2, Maximize2, Minimize2, XIcon } from "lucide-react";
import { useRef, useState, type FormEvent } from "react";

import { Button } from "@/components/ui/button";
import { Dialog, DialogClose, DialogContent, DialogDescription, DialogTitle } from "@/components/ui/dialog";
import { InputGroup, InputGroupAddon } from "@/components/ui/input-group";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import type { UploadedWorkOrderFile } from "@/hooks/useWorkOrderFileUpload";
import { cn } from "@/lib/utils";

import { CreateWorkOrderRequestAttachButton } from "./CreateWorkOrderRequestAttachButton";
import { CreateWorkOrderRequestAttachments } from "./CreateWorkOrderRequestAttachments";
import { CREATE_WORK_ORDER_REQUEST_COPY } from "./createWorkOrderRequestCopy";
import { createWorkOrderRequestImages, mergeCreateWorkOrderRequestImages } from "./lib/createWorkOrderRequestImages";
import { MAX_DERIVED_WORK_ORDER_TITLE_LENGTH } from "./lib/derivedWorkOrderTitle";
import { useCreateWorkOrderRequestForm } from "./useCreateWorkOrderRequestForm";
import { WorkOrderDescriptionEditor } from "./WorkOrderDescriptionEditor";

export interface CreateWorkOrderRequestDraft {
  title: string;
  description: string;
}

export interface CreateWorkOrderRequestDialogProps {
  open: boolean;
  description: string;
  maxLength: number;
  isCreating?: boolean;
  isUploading?: boolean;
  fileUrls?: Record<string, string>;
  onClose: () => void;
  onDescriptionChange: (next: string) => void;
  onCreate: (draft: CreateWorkOrderRequestDraft) => void;
  onUploadFiles?: (files: FileList | File[]) => Promise<UploadedWorkOrderFile[]>;
  initialAttachedFiles?: UploadedWorkOrderFile[];
}

export function CreateWorkOrderRequestDialog({
  open,
  description,
  maxLength,
  isCreating = false,
  isUploading = false,
  fileUrls,
  onClose,
  onDescriptionChange,
  onCreate,
  onUploadFiles,
  initialAttachedFiles = [],
}: CreateWorkOrderRequestDialogProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const contentRef = useRef<HTMLDivElement>(null);
  const form = useCreateWorkOrderRequestForm({
    open,
    description,
    isCreating,
    isUploading,
    onDescriptionChange,
    onCreate,
    onUploadFiles,
    initialAttachedFiles,
  });
  const attachedImages = mergeCreateWorkOrderRequestImages(
    createWorkOrderRequestImages(description, fileUrls),
    form.attachedFiles,
  );

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen && !isCreating && !isUploading) {
          onClose();
        }
      }}
    >
      <DialogContent
        ref={contentRef}
        showCloseButton={false}
        size={isExpanded ? "90vw" : "large"}
        className={cn(
          "create-work-order-request-attachments flex min-h-0 flex-col gap-0 overflow-hidden rounded-2xl p-0",
          isExpanded
            ? "max-h-[90vh] sm:max-w-none"
            : "h-auto max-h-[min(36rem,calc(100dvh-2rem))] w-[min(32rem,calc(100%-2rem))] max-w-[32rem] sm:max-w-[32rem]",
        )}
        data-testid="create-work-order-request-dialog"
        onOpenAutoFocus={(event) => {
          event.preventDefault();
          const description = contentRef.current?.querySelector<HTMLElement>("#work-order-description-input");
          description?.focus();
        }}
      >
        <DialogTitle className="sr-only">{CREATE_WORK_ORDER_REQUEST_COPY.title}</DialogTitle>
        <DialogDescription className="sr-only">{CREATE_WORK_ORDER_REQUEST_COPY.description}</DialogDescription>
        <RequestDialogChrome isExpanded={isExpanded} onToggleExpanded={() => setIsExpanded((current) => !current)} />
        <form
          className="flex min-h-0 flex-1 flex-col overflow-visible"
          onSubmit={(event: FormEvent) => {
            event.preventDefault();
            form.submitDraft();
          }}
        >
          <RequestDialogTitleField
            busy={form.busy}
            derivedTitle={form.derivedTitle}
            titleDirty={form.titleDirty}
            titleValue={form.titleValue}
            onTitleChange={form.handleTitleChange}
          />
          <Label htmlFor="work-order-description-input" className="sr-only">
            {CREATE_WORK_ORDER_REQUEST_COPY.placeholder}
          </Label>
          <div
            className="min-h-0 flex-1 overflow-x-hidden overflow-y-auto overscroll-contain [scrollbar-gutter:stable] [scrollbar-width:thin]"
            data-testid="create-work-order-request-body"
          >
            <div className="px-4 pt-2 pb-1">
              <WorkOrderDescriptionEditor
                value={description}
                maxLength={maxLength}
                disabled={form.busy}
                autoFocus
                placeholder={CREATE_WORK_ORDER_REQUEST_COPY.placeholder}
                className="min-h-[6.5rem] max-w-full text-[14px] leading-6 [&_.ProseMirror]:max-w-full [&_.work-order-file-image]:my-2 [&_.work-order-file-image]:max-h-40 [&_.work-order-file-image]:w-auto [&_.work-order-file-image]:max-w-full [&_.work-order-file-image]:object-contain"
                fileUrls={fileUrls}
                onUploadFiles={form.uploadAcceptedFiles}
                isUploading={isUploading}
                canRemoveImages
                onChange={onDescriptionChange}
              />
            </div>
          </div>
          <RequestDialogFooter
            attachedImages={attachedImages}
            canAttach={Boolean(onUploadFiles) && form.canAttach}
            canCreate={form.canCreate}
            isCreating={isCreating}
            showAttach={Boolean(onUploadFiles)}
            onAttach={(files) => void form.handleAttach(files)}
            onRemoveAttachment={form.handleRemoveAttachment}
          />
        </form>
      </DialogContent>
    </Dialog>
  );
}

function RequestDialogChrome({ isExpanded, onToggleExpanded }: { isExpanded: boolean; onToggleExpanded: () => void }) {
  return (
    <div className="absolute top-2.5 right-2.5 z-10 flex items-center gap-0.5">
      <Button
        type="button"
        variant="ghost"
        size="icon"
        className="size-7 text-muted-foreground"
        aria-label={isExpanded ? CREATE_WORK_ORDER_REQUEST_COPY.collapse : CREATE_WORK_ORDER_REQUEST_COPY.expand}
        data-testid="create-work-order-request-fullscreen"
        onClick={onToggleExpanded}
      >
        {isExpanded ? <Minimize2 className="size-3.5" aria-hidden /> : <Maximize2 className="size-3.5" aria-hidden />}
      </Button>
      <DialogClose
        className="flex size-7 cursor-pointer items-center justify-center rounded-full text-muted-foreground hover:bg-accent focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
        data-testid="create-work-order-request-close"
      >
        <XIcon className="size-4" />
        <span className="sr-only">Close</span>
      </DialogClose>
    </div>
  );
}

function RequestDialogTitleField({
  busy,
  derivedTitle,
  titleDirty,
  titleValue,
  onTitleChange,
}: {
  busy: boolean;
  derivedTitle: string;
  titleDirty: boolean;
  titleValue: string;
  onTitleChange: (next: string) => void;
}) {
  return (
    <div className="shrink-0 px-4 pt-4 pr-20">
      <Label htmlFor="create-work-order-request-title" className="sr-only">
        {CREATE_WORK_ORDER_REQUEST_COPY.titleField}
      </Label>
      <Textarea
        id="create-work-order-request-title"
        data-testid="create-work-order-request-title"
        value={titleValue}
        maxLength={MAX_DERIVED_WORK_ORDER_TITLE_LENGTH}
        disabled={busy}
        rows={1}
        placeholder={derivedTitle || CREATE_WORK_ORDER_REQUEST_COPY.titlePlaceholder}
        onChange={(event) => onTitleChange(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Enter" && !event.metaKey && !event.ctrlKey) {
            event.preventDefault();
          }
        }}
        className={cn(
          "min-h-0 resize-none border-0 bg-transparent p-0 text-[16px] font-medium shadow-none focus-visible:ring-0",
          titleDirty ? "text-foreground" : "text-muted-foreground",
        )}
      />
    </div>
  );
}

function RequestDialogFooter({
  attachedImages,
  canAttach,
  canCreate,
  isCreating,
  showAttach,
  onAttach,
  onRemoveAttachment,
}: {
  attachedImages: ReturnType<typeof mergeCreateWorkOrderRequestImages>;
  canAttach: boolean;
  canCreate: boolean;
  isCreating: boolean;
  showAttach: boolean;
  onAttach: (files: FileList | File[]) => void;
  onRemoveAttachment: (id: string) => void;
}) {
  return (
    <InputGroup className="h-auto shrink-0 overflow-visible border-0 bg-transparent shadow-none dark:bg-transparent">
      <InputGroupAddon align="block-end" className="items-end justify-between gap-3 overflow-visible px-3 pt-1 pb-3">
        <div className="flex min-w-0 items-end gap-2 overflow-visible">
          {showAttach ? <CreateWorkOrderRequestAttachButton disabled={!canAttach} onAttach={onAttach} /> : <span />}
          {attachedImages.length > 0 ? (
            <CreateWorkOrderRequestAttachments images={attachedImages} onRemove={onRemoveAttachment} />
          ) : null}
        </div>
        <Button
          type="submit"
          size="icon"
          className="size-8 rounded-full"
          disabled={!canCreate}
          aria-label={isCreating ? CREATE_WORK_ORDER_REQUEST_COPY.creating : CREATE_WORK_ORDER_REQUEST_COPY.create}
          aria-keyshortcuts="Meta+Enter Control+Enter"
          data-testid="create-work-order-request-create"
        >
          {isCreating ? (
            <Loader2 className="size-3.5 animate-spin" aria-hidden />
          ) : (
            <ArrowUp className="size-3.5" aria-hidden />
          )}
        </Button>
      </InputGroupAddon>
    </InputGroup>
  );
}
