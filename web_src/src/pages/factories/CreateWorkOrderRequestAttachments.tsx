import { Trash2, X } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";

import { Button } from "@/components/ui/button";
import type { UploadedWorkOrderFile } from "@/hooks/useWorkOrderFileUpload";
import { HostedVideoEmbed } from "@/pages/app/HostedVideoEmbed";

import { CREATE_WORK_ORDER_REQUEST_COPY } from "./createWorkOrderRequestCopy";
import type { CreateWorkOrderRequestImage } from "./lib/createWorkOrderRequestImages";
import { PendingWorkOrderFileChips } from "./PendingWorkOrderFileChips";

import "./createWorkOrderRequestAttachments.css";

export interface CreateWorkOrderRequestAttachmentsProps {
  images: CreateWorkOrderRequestImage[];
  onRemove?: (id: string) => void;
}

export function CreateWorkOrderRequestPreviewRow({
  images,
  files,
  onRemove,
}: {
  images: CreateWorkOrderRequestImage[];
  files: UploadedWorkOrderFile[];
  onRemove: (id: string) => void;
}) {
  if (images.length === 0 && files.length === 0) {
    return null;
  }

  return (
    <div
      className="flex w-full flex-wrap items-start justify-start gap-2"
      data-testid="create-work-order-request-preview-row"
    >
      {images.length > 0 ? <CreateWorkOrderRequestAttachments images={images} onRemove={onRemove} /> : null}
      {files.length > 0 ? <PendingWorkOrderFileChips files={files} onRemove={onRemove} /> : null}
    </div>
  );
}

export function CreateWorkOrderRequestAttachments({ images, onRemove }: CreateWorkOrderRequestAttachmentsProps) {
  const [expanded, setExpanded] = useState<CreateWorkOrderRequestImage | null>(null);

  useEffect(() => {
    if (!expanded) {
      return;
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape") {
        return;
      }
      event.preventDefault();
      event.stopPropagation();
      setExpanded(null);
    };
    window.addEventListener("keydown", onKeyDown, true);
    return () => window.removeEventListener("keydown", onKeyDown, true);
  }, [expanded]);

  if (images.length === 0) {
    return null;
  }

  return (
    <>
      <div
        className="flex min-w-0 flex-wrap items-start gap-2"
        aria-label={CREATE_WORK_ORDER_REQUEST_COPY.attachedImages}
        data-testid="create-work-order-request-attachments"
      >
        {images.map((image) => (
          <AttachmentTile key={image.id} image={image} onOpen={() => setExpanded(image)} onRemove={onRemove} />
        ))}
      </div>
      {expanded ? (
        <RequestImageExpand
          image={expanded}
          onClose={() => setExpanded(null)}
          onRemove={
            onRemove
              ? () => {
                  onRemove(expanded.id);
                  setExpanded(null);
                }
              : undefined
          }
        />
      ) : null}
    </>
  );
}

function AttachmentTile({
  image,
  onOpen,
  onRemove,
}: {
  image: CreateWorkOrderRequestImage;
  onOpen: () => void;
  onRemove?: (id: string) => void;
}) {
  const label = image.alt || image.id;

  return (
    <div className="group relative size-14 shrink-0 overflow-hidden rounded-md border border-slate-200 bg-slate-50 dark:border-gray-700 dark:bg-gray-900">
      <Button
        type="button"
        variant="ghost"
        className="relative block size-full h-auto cursor-pointer rounded-none border-0 bg-transparent p-0 shadow-none hover:bg-transparent dark:hover:bg-transparent"
        aria-label={`${CREATE_WORK_ORDER_REQUEST_COPY.openImage}: ${label}`}
        data-testid={`create-work-order-request-attachment-${image.id}`}
        onClick={onOpen}
      >
        <AttachmentTileMedia image={image} />
      </Button>
      {onRemove ? (
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label={`Remove ${label}`}
          data-testid={`create-work-order-request-tile-remove-${image.id}`}
          className="absolute top-0.5 right-0.5 z-10 size-4 rounded-full bg-slate-900/70 text-white opacity-0 transition-opacity group-hover:opacity-100 hover:bg-slate-900/70 hover:text-white focus-visible:opacity-100 dark:bg-slate-900/70 dark:text-white dark:hover:bg-slate-900/70 dark:hover:text-white"
          onClick={(event) => {
            event.stopPropagation();
            onRemove(image.id);
          }}
        >
          <X className="size-3" aria-hidden />
        </Button>
      ) : null}
    </div>
  );
}

function AttachmentTileMedia({ image }: { image: CreateWorkOrderRequestImage }) {
  if (image.hostedVideo || image.isAudio || !image.src) {
    return <AttachmentTypeLabel label={attachmentTypeLabel(image)} />;
  }
  if (image.isVideo) {
    return <VideoAttachmentPreview src={image.src} />;
  }
  return <img src={image.src} alt="" className="pointer-events-none size-full object-cover" />;
}

function VideoAttachmentPreview({ src }: { src: string }) {
  const [frameReady, setFrameReady] = useState(false);

  return (
    <>
      <video
        src={src}
        muted
        playsInline
        preload="auto"
        className={
          frameReady ? "pointer-events-none size-full object-cover" : "pointer-events-none absolute size-0 opacity-0"
        }
        onLoadedData={() => setFrameReady(true)}
        onError={() => setFrameReady(false)}
      />
      {frameReady ? null : <AttachmentTypeLabel label="Video" />}
    </>
  );
}

function attachmentTypeLabel(image: CreateWorkOrderRequestImage): string {
  if (image.hostedVideo) {
    return image.hostedVideo.providerName;
  }
  if (image.isAudio) {
    return "Audio";
  }
  if (image.isVideo) {
    return "Video";
  }
  return image.alt || "Image";
}

function AttachmentTypeLabel({ label }: { label: string }) {
  return (
    <span className="flex size-full items-center justify-center px-1 text-center text-[10px] leading-tight font-medium text-muted-foreground">
      {label}
    </span>
  );
}

function RequestImageExpand({
  image,
  onClose,
  onRemove,
}: {
  image: CreateWorkOrderRequestImage;
  onClose: () => void;
  onRemove?: () => void;
}) {
  const dialogRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const previous = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const root = dialogRef.current;
    if (!root) {
      return;
    }

    const focusables = () => [...root.querySelectorAll<HTMLButtonElement>("button")].filter((node) => !node.disabled);
    root.querySelector<HTMLElement>("[data-testid='create-work-order-request-image-close']")?.focus();

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Tab") {
        return;
      }
      const items = focusables();
      if (items.length === 0) {
        event.preventDefault();
        return;
      }
      const first = items[0];
      const last = items[items.length - 1];
      if (event.shiftKey && document.activeElement === first) {
        event.preventDefault();
        last.focus();
        return;
      }
      if (!event.shiftKey && document.activeElement === last) {
        event.preventDefault();
        first.focus();
      }
    };

    window.addEventListener("keydown", onKeyDown, true);
    return () => {
      window.removeEventListener("keydown", onKeyDown, true);
      previous?.focus();
    };
  }, []);

  return createPortal(
    <div className="create-work-order-request-attachments t-resize-portal" style={{ pointerEvents: "auto" }}>
      <div className="t-resize-backdrop" data-testid="create-work-order-request-image-backdrop" onClick={onClose}>
        <div
          ref={dialogRef}
          className="t-resize is-open"
          role="dialog"
          aria-modal="true"
          aria-label={image.alt || CREATE_WORK_ORDER_REQUEST_COPY.openImage}
          data-testid="create-work-order-request-image-expand"
        >
          <div className="t-resize-actions">
            {onRemove ? (
              <button
                type="button"
                className="t-resize-action"
                aria-label={CREATE_WORK_ORDER_REQUEST_COPY.deleteImage}
                data-testid="create-work-order-request-image-remove"
                onClick={(event) => {
                  event.stopPropagation();
                  onRemove();
                }}
              >
                <Trash2 className="size-3.5" aria-hidden />
              </button>
            ) : null}
            <button
              type="button"
              className="t-resize-action"
              aria-label={CREATE_WORK_ORDER_REQUEST_COPY.closeImage}
              data-testid="create-work-order-request-image-close"
              onClick={(event) => {
                event.stopPropagation();
                onClose();
              }}
            >
              <X className="size-3.5" aria-hidden />
            </button>
          </div>
          {image.hostedVideo ? (
            <HostedVideoEmbed video={image.hostedVideo} className="t-resize-img" />
          ) : image.isAudio ? (
            <audio className="t-resize-img" src={image.src} controls aria-label={image.alt} />
          ) : image.isVideo ? (
            <video className="t-resize-img" src={image.src} controls playsInline aria-label={image.alt} />
          ) : (
            <img className="t-resize-img" src={image.src} alt={image.alt} />
          )}
        </div>
      </div>
    </div>,
    document.body,
  );
}
