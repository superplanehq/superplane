import { Trash2, X } from "lucide-react";
import { useEffect, useRef, useState, type CSSProperties } from "react";
import { createPortal } from "react-dom";

import { CREATE_WORK_ORDER_REQUEST_COPY } from "./createWorkOrderRequestCopy";
import type { CreateWorkOrderRequestImage } from "./lib/createWorkOrderRequestImages";

import "./createWorkOrderRequestAttachments.css";

const STACK_CARD_REM = 3.5;
const STACK_HOVER_MAX_REM = 14;
const STACK_HOVER_STEP_REM = 3.75;
const STACK_HOVER_MAX_SPREAD_REM = STACK_HOVER_MAX_REM - STACK_CARD_REM;

function stackSlot(index: number, count: number): CSSProperties {
  const sign = index % 2 === 0 ? 1 : -1;
  const step = count > 1 ? Math.min(STACK_HOVER_STEP_REM, STACK_HOVER_MAX_SPREAD_REM / (count - 1)) : 0;
  return {
    "--cx": `${6 + index * 2}px`,
    "--cy": `${6 + (index % 2) * 4}px`,
    "--rot": `${sign * Math.min(4 + index * 2, 12)}deg`,
    "--hx": `${index * step}rem`,
    "--hy": "8px",
    zIndex: index,
  } as CSSProperties;
}

export interface CreateWorkOrderRequestAttachmentsProps {
  images: CreateWorkOrderRequestImage[];
  onRemove?: (id: string) => void;
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
        className="t-stack"
        style={{ "--stack-n": images.length } as CSSProperties}
        aria-label={CREATE_WORK_ORDER_REQUEST_COPY.attachedImages}
        data-testid="create-work-order-request-attachments"
      >
        {images.map((image, index) => (
          <button
            key={image.id}
            type="button"
            className="t-stack-card"
            style={stackSlot(index, images.length)}
            aria-label={`${CREATE_WORK_ORDER_REQUEST_COPY.openImage}: ${image.alt || image.id}`}
            data-testid={`create-work-order-request-attachment-${image.id}`}
            onClick={() => setExpanded(image)}
          >
            {image.isVideo ? <video src={image.src} muted playsInline /> : <img src={image.src} alt="" />}
          </button>
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
          {image.isVideo ? (
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
