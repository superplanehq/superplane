import { Trash2, X } from "lucide-react";
import { useCallback, useEffect, useState, type CSSProperties } from "react";
import { createPortal } from "react-dom";

import { CREATE_WORK_ORDER_REQUEST_COPY } from "./createWorkOrderRequestCopy";
import type { CreateWorkOrderRequestImage } from "./lib/createWorkOrderRequestImages";

import "./createWorkOrderRequestAttachments.css";

interface StackSlot {
  cx: string;
  cy: string;
  rot: string;
  hx: string;
  hy: string;
}

function stackSlot(index: number): StackSlot {
  const sign = index % 2 === 0 ? 1 : -1;
  return {
    cx: `${6 + index * 2}px`,
    cy: `${6 + (index % 2) * 4}px`,
    rot: `${sign * Math.min(4 + index * 2, 12)}deg`,
    hx: `${index * 3.75}rem`,
    hy: "8px",
  };
}

export interface CreateWorkOrderRequestAttachmentsProps {
  images: CreateWorkOrderRequestImage[];
  onRemove?: (id: string) => void;
}

export function CreateWorkOrderRequestAttachments({ images, onRemove }: CreateWorkOrderRequestAttachmentsProps) {
  const [expanded, setExpanded] = useState<CreateWorkOrderRequestImage | null>(null);
  const [isOpen, setIsOpen] = useState(false);
  const [isClosing, setIsClosing] = useState(false);

  const closeExpanded = useCallback(() => {
    if (!expanded || isClosing) {
      return;
    }
    setIsOpen(false);
    setIsClosing(true);
  }, [expanded, isClosing]);

  useEffect(() => {
    if (!expanded || isClosing) {
      return;
    }
    const frame = requestAnimationFrame(() => {
      requestAnimationFrame(() => setIsOpen(true));
    });
    return () => cancelAnimationFrame(frame);
  }, [expanded, isClosing]);

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
      closeExpanded();
    };
    window.addEventListener("keydown", onKeyDown, true);
    return () => window.removeEventListener("keydown", onKeyDown, true);
  }, [closeExpanded, expanded]);

  useEffect(() => {
    if (!isClosing) {
      return;
    }
    const delayMs = window.matchMedia("(prefers-reduced-motion: reduce)").matches ? 0 : 300;
    const timer = window.setTimeout(() => {
      setExpanded(null);
      setIsClosing(false);
    }, delayMs);
    return () => window.clearTimeout(timer);
  }, [isClosing]);

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
        {images.map((image, index) => {
          const slot = stackSlot(index);
          return (
            <button
              key={image.id}
              type="button"
              className="t-stack-card"
              style={
                {
                  "--cx": slot.cx,
                  "--cy": slot.cy,
                  "--rot": slot.rot,
                  "--hx": slot.hx,
                  "--hy": slot.hy,
                  zIndex: index,
                } as CSSProperties
              }
              aria-label={`${CREATE_WORK_ORDER_REQUEST_COPY.openImage}: ${image.alt || image.id}`}
              data-testid={`create-work-order-request-attachment-${image.id}`}
              onClick={() => {
                setIsClosing(false);
                setIsOpen(false);
                setExpanded(image);
              }}
            >
              <img src={image.src} alt="" />
            </button>
          );
        })}
      </div>
      {expanded ? (
        <RequestImageExpand
          image={expanded}
          isOpen={isOpen}
          onClose={closeExpanded}
          onDismiss={() => {
            setExpanded(null);
            setIsOpen(false);
            setIsClosing(false);
          }}
          onRemove={
            onRemove
              ? () => {
                  onRemove(expanded.id);
                  setExpanded(null);
                  setIsOpen(false);
                  setIsClosing(false);
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
  isOpen,
  onClose,
  onDismiss,
  onRemove,
}: {
  image: CreateWorkOrderRequestImage;
  isOpen: boolean;
  onClose: () => void;
  onDismiss: () => void;
  onRemove?: () => void;
}) {
  return createPortal(
    <div className="create-work-order-request-attachments t-resize-portal" style={{ pointerEvents: "auto" }}>
      <div className="t-resize-backdrop" data-testid="create-work-order-request-image-backdrop" onClick={onClose}>
        <div className={`t-resize${isOpen ? " is-open" : ""}`} data-testid="create-work-order-request-image-expand">
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
                onDismiss();
              }}
            >
              <X className="size-3.5" aria-hidden />
            </button>
          </div>
          <img className="t-resize-img" src={image.src} alt={image.alt} />
        </div>
      </div>
    </div>,
    document.body,
  );
}
