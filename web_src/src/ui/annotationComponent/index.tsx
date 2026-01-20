import React, { useEffect, useLayoutEffect, useMemo } from "react";
import { Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { SelectionWrapper } from "../selectionWrapper";
import { setActiveNoteId } from "./noteFocus";
import { ComponentActionsProps } from "../types/componentActions";

type AnnotationColor = "yellow" | "blue" | "green" | "purple";

const NOTE_COLORS: Record<AnnotationColor, { label: string; container: string; background: string; dot: string }> = {
  yellow: {
    label: "Yellow",
    container: "bg-yellow-100",
    background: "bg-yellow-100",
    dot: "bg-yellow-200 border-yellow-500",
  },
  blue: {
    label: "Sky",
    container: "bg-sky-100",
    background: "bg-sky-100",
    dot: "bg-sky-200 border-sky-500",
  },
  green: {
    label: "Green",
    container: "bg-green-100",
    background: "bg-green-100",
    dot: "bg-green-200 border-green-500",
  },
  purple: {
    label: "Purple",
    container: "bg-purple-100",
    background: "bg-purple-100",
    dot: "bg-purple-200 border-purple-500",
  },
};

const noteDrafts = new Map<string, string>();

export interface AnnotationComponentProps extends ComponentActionsProps {
  title: string;
  annotationText?: string;
  annotationColor?: AnnotationColor;
  noteId?: string;
  selected?: boolean;
  hideActionsButton?: boolean;
  onAnnotationUpdate?: (updates: { text?: string; color?: AnnotationColor }) => void;
}

const AnnotationComponentBase: React.FC<AnnotationComponentProps> = ({
  title,
  annotationText = "",
  annotationColor = "yellow",
  noteId,
  selected = false,
  onDelete,
  hideActionsButton,
  onAnnotationUpdate,
}) => {
  const textareaRef = React.useRef<HTMLTextAreaElement | null>(null);
  const containerRef = React.useRef<HTMLDivElement | null>(null);
  const lastPointerDownOutsideRef = React.useRef(false);

  const syncTextareaHeight = () => {
    const textarea = textareaRef.current;
    if (!textarea) return;
    textarea.style.height = "auto";
    textarea.style.height = `${textarea.scrollHeight}px`;
  };

  useEffect(() => {
    if (!noteId) return;
    const textarea = textareaRef.current;
    if (!textarea) return;
    const nextValue = noteDrafts.get(noteId) ?? annotationText ?? "";
    if (textarea.value !== nextValue) {
      textarea.value = nextValue;
    }
    noteDrafts.set(noteId, nextValue);
    requestAnimationFrame(syncTextareaHeight);
  }, [annotationText, noteId]);

  useLayoutEffect(() => {
    if (!noteId) return;
    const textarea = textareaRef.current;
    if (!textarea) return;
    if (!noteDrafts.has(noteId)) {
      noteDrafts.set(noteId, annotationText || "");
    }
    const draft = noteDrafts.get(noteId) ?? "";
    if (textarea.value !== draft) {
      textarea.value = draft;
      syncTextareaHeight();
    }
  }, [noteId, annotationText]);

  useEffect(() => {
    const handlePointerDown = (event: PointerEvent) => {
      const target = event.target as Node | null;
      if (!containerRef.current || !target) {
        lastPointerDownOutsideRef.current = true;
        return;
      }
      lastPointerDownOutsideRef.current = !containerRef.current.contains(target);
    };

    document.addEventListener("pointerdown", handlePointerDown, true);
    return () => {
      document.removeEventListener("pointerdown", handlePointerDown, true);
    };
  }, []);

  const activeColor = annotationColor && NOTE_COLORS[annotationColor] ? annotationColor : "yellow";
  const colorStyles = NOTE_COLORS[activeColor];

  const handleTextCommit = () => {
    const nextText = textareaRef.current?.value ?? "";
    if (nextText !== (annotationText || "")) {
      onAnnotationUpdate?.({ text: nextText });
    }
  };

  const colorOptions = useMemo(
    () =>
      (Object.keys(NOTE_COLORS) as AnnotationColor[]).map((value) => ({
        value,
        dot: NOTE_COLORS[value].dot,
      })),
    [],
  );

  return (
    <SelectionWrapper selected={selected}>
      <div
        ref={containerRef}
        className={cn(
          "group relative flex w-[20rem] flex-col rounded-md shadow-md outline outline-gray-950/10",
          colorStyles.container,
        )}
      >
        <div className={cn("canvas-node-drag-handle h-5 w-full rounded-t-md cursor-grab", colorStyles.background)}>
          <div className="flex h-full w-full flex-col items-stretch justify-center gap-0.5 px-2">
            <span className="h-px w-full bg-black/15" />
            <span className="h-px w-full bg-black/15" />
            <span className="h-px w-full bg-black/15" />
          </div>
        </div>

        {!hideActionsButton && (
          <>
            <div className="absolute -top-12 right-0 z-10 h-12 w-44 opacity-0" />
            <div className="absolute -top-8 right-0 z-10 hidden items-center gap-2 group-hover:flex nodrag">
              <div className="group/swatch relative flex items-center px-0.5 py-0.5">
                <button
                  type="button"
                  onClick={(event) => {
                    event.preventDefault();
                    event.stopPropagation();
                  }}
                  className={cn("h-4 w-4 rounded-full border transition", NOTE_COLORS[activeColor].dot)}
                  aria-label={`Current note color: ${NOTE_COLORS[activeColor].label}`}
                />
                <div className="absolute right-0 top-1/2 hidden -translate-y-1/2 items-center gap-2 pr-0.5 group-hover/swatch:flex">
                  {colorOptions.map((option) => (
                    <button
                      key={option.value}
                      type="button"
                      onClick={(event) => {
                        event.preventDefault();
                        event.stopPropagation();
                        onAnnotationUpdate?.({ color: option.value });
                      }}
                      className={cn("h-4 w-4 rounded-full border transition", option.dot)}
                      aria-label={`Set note color to ${NOTE_COLORS[option.value].label}`}
                    />
                  ))}
                </div>
              </div>
              {onDelete && (
                <button
                type="button"
                onClick={(event) => {
                  event.preventDefault();
                  event.stopPropagation();
                  onDelete();
                }}
                className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800"
                aria-label="Delete note"
              >
                <Trash2 size={16} />
                </button>
              )}
            </div>
          </>
        )}

        <div className="px-3 pb-3">
          <textarea
            ref={textareaRef}
            data-note-id={noteId || undefined}
            defaultValue={noteId ? (noteDrafts.get(noteId) ?? annotationText) : annotationText}
            onInput={(event) => {
              const value = (event.target as HTMLTextAreaElement).value;
              if (noteId) {
                noteDrafts.set(noteId, value);
                setActiveNoteId(noteId);
              }
              syncTextareaHeight();
              if (onAnnotationUpdate) {
                onAnnotationUpdate({ text: value });
              }
            }}
            onBlur={() => {
              handleTextCommit();
              const activeElement = document.activeElement as Node | null;
              const shouldRestore =
                !lastPointerDownOutsideRef.current && (!activeElement || activeElement === document.body);
              if (shouldRestore) {
                requestAnimationFrame(() => textareaRef.current?.focus());
              } else if (noteId) {
                setActiveNoteId(null);
              }
            }}
            onFocus={() => {
              if (noteId) {
                setActiveNoteId(noteId);
              }
              lastPointerDownOutsideRef.current = false;
            }}
            className={cn(
              "nodrag min-h-[120px] w-full resize-none bg-transparent text-sm leading-normal outline-none",
              "text-gray-800",
              "placeholder:text-black/50",
            )}
            placeholder="Start typing..."
            aria-label={`${title} note`}
          />
        </div>
      </div>
    </SelectionWrapper>
  );
};

export const AnnotationComponent = React.memo(
  AnnotationComponentBase,
  (prev, next) =>
    prev.title === next.title &&
    prev.annotationText === next.annotationText &&
    prev.annotationColor === next.annotationColor &&
    prev.selected === next.selected &&
    prev.hideActionsButton === next.hideActionsButton,
);
