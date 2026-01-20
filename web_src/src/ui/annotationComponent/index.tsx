import React, { useEffect, useLayoutEffect, useMemo, useState } from "react";
import { EllipsisVertical, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";
import { SelectionWrapper } from "../selectionWrapper";
import { setActiveNoteId } from "./noteFocus";
import { ComponentActionsProps } from "../types/componentActions";

type AnnotationColor = "yellow" | "blue" | "green" | "purple";

const NOTE_COLORS: Record<
  AnnotationColor,
  { label: string; container: string; background: string; dot: string; text: string; placeholder: string }
> = {
  yellow: {
    label: "Yellow",
    container: "bg-yellow-100",
    background: "bg-yellow-100",
    dot: "bg-yellow-200 border-yellow-300",
    text: "text-yellow-900",
    placeholder: "placeholder:text-yellow-700/60",
  },
  blue: {
    label: "Sky",
    container: "bg-sky-100",
    background: "bg-sky-100",
    dot: "bg-sky-200 border-sky-300",
    text: "text-sky-900",
    placeholder: "placeholder:text-sky-700/60",
  },
  green: {
    label: "Green",
    container: "bg-green-100",
    background: "bg-green-100",
    dot: "bg-green-200 border-green-300",
    text: "text-green-900",
    placeholder: "placeholder:text-green-700/60",
  },
  purple: {
    label: "Purple",
    container: "bg-purple-100",
    background: "bg-purple-100",
    dot: "bg-purple-200 border-purple-300",
    text: "text-purple-900",
    placeholder: "placeholder:text-purple-700/60",
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
  width?: number;
  height?: number;
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
  width = 320,
  height = 170,
}) => {
  const [isMenuOpen, setIsMenuOpen] = useState(false);
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
          "group relative flex flex-col rounded-md shadow-md outline outline-gray-950/10",
          colorStyles.container,
        )}
        style={{ width: width, height: height }}
      >
        <div className={cn("canvas-node-drag-handle h-5 w-full rounded-t-md cursor-grab", colorStyles.background)}>
          <div className="flex h-full w-full flex-col items-stretch justify-center gap-0.5 px-2">
            <span className="h-px w-full bg-black/15" />
            <span className="h-px w-full bg-black/15" />
            <span className="h-px w-full bg-black/15" />
          </div>
        </div>

        {!hideActionsButton && (
          <div className="absolute top-0 -right-7 nodrag">
            <DropdownMenu open={isMenuOpen} onOpenChange={setIsMenuOpen}>
              <DropdownMenuTrigger asChild>
                <button
                  type="button"
                  className={cn(
                    "flex h-6 w-6 items-center justify-center rounded border border-transparent text-slate-600 opacity-0 transition group-hover:opacity-100 hover:bg-gray-950/10 hover:text-slate-800",
                    isMenuOpen && "opacity-100",
                  )}
                  aria-label="Note actions"
                >
                  <EllipsisVertical size={16} />
                </button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" sideOffset={6} className="w-44">
                <DropdownMenuRadioGroup
                  value={activeColor}
                  onValueChange={(value) => onAnnotationUpdate?.({ color: value as AnnotationColor })}
                  className="flex items-center gap-3 px-2 py-2"
                >
                  {colorOptions.map((option) => (
                    <DropdownMenuRadioItem
                      key={option.value}
                      value={option.value}
                      className="h-6 w-6 justify-center p-0 data-[state=checked]:ring-2 data-[state=checked]:ring-sky-500 data-[state=checked]:ring-offset-4 data-[state=checked]:ring-offset-white [&>span:first-child]:hidden rounded-full"
                      onSelect={(event) => {
                        event.preventDefault();
                      }}
                    >
                      <span className={cn("h-6 w-6 rounded-full border", option.dot)} />
                    </DropdownMenuRadioItem>
                  ))}
                </DropdownMenuRadioGroup>
                {onDelete && (
                  <>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem
                      onSelect={(event) => {
                        event.preventDefault();
                        onDelete?.();
                      }}
                      className="cursor-pointer"
                    >
                      <Trash2 size={16} />
                      Delete Note
                    </DropdownMenuItem>
                  </>
                )}
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        )}

        <div className="px-3 pb-3 flex-1 flex flex-col">
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
              "nodrag flex-1 w-full resize-none bg-transparent text-sm leading-normal outline-none",
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
    prev.hideActionsButton === next.hideActionsButton &&
    prev.width === next.width &&
    prev.height === next.height,
);
