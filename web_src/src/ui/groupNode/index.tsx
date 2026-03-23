import React, { useCallback, useEffect, useMemo, useState } from "react";
import debounce from "lodash.debounce";
import { Trash2, Ungroup } from "lucide-react";
import { cn } from "@/lib/utils";

export type GroupColor = "gray" | "blue" | "green" | "purple";

const GROUP_COLORS: Record<
  GroupColor,
  { label: string; border: string; background: string; dot: string; text: string }
> = {
  gray: {
    label: "Gray",
    border: "border-slate-300",
    background: "bg-slate-50/80",
    dot: "bg-slate-200 border-slate-500",
    text: "text-slate-500",
  },
  blue: {
    label: "Blue",
    border: "border-sky-300",
    background: "bg-sky-50/80",
    dot: "bg-sky-200 border-sky-500",
    text: "text-sky-600",
  },
  green: {
    label: "Green",
    border: "border-green-300",
    background: "bg-green-50/80",
    dot: "bg-green-200 border-green-500",
    text: "text-green-600",
  },
  purple: {
    label: "Purple",
    border: "border-purple-300",
    background: "bg-purple-50/80",
    dot: "bg-purple-200 border-purple-500",
    text: "text-purple-600",
  },
};

export interface GroupNodeProps {
  groupLabel?: string;
  groupColor?: GroupColor;
  selected?: boolean;
  hideActionsButton?: boolean;
  onGroupUpdate?: (updates: { label?: string; color?: GroupColor }) => void;
  onUngroup?: () => void;
  onDelete?: () => void;
}

const GroupNodeBase: React.FC<GroupNodeProps> = ({
  groupLabel = "Group",
  groupColor = "gray",
  selected = false,
  hideActionsButton,
  onGroupUpdate,
  onUngroup,
  onDelete,
}) => {
  const inputRef = React.useRef<HTMLInputElement | null>(null);

  const [isEditingLabel, setIsEditingLabel] = useState(false);
  const [localLabel, setLocalLabel] = useState(groupLabel);

  useEffect(() => {
    setLocalLabel(groupLabel);
  }, [groupLabel]);

  const activeColor = groupColor && GROUP_COLORS[groupColor] ? groupColor : "gray";
  const colorStyles = GROUP_COLORS[activeColor];

  const onGroupUpdateRef = React.useRef(onGroupUpdate);
  useEffect(() => {
    onGroupUpdateRef.current = onGroupUpdate;
  }, [onGroupUpdate]);

  const debouncedLabelUpdate = useMemo(
    () =>
      debounce((nextLabel: string) => {
        onGroupUpdateRef.current?.({ label: nextLabel });
      }, 500),
    [],
  );

  useEffect(() => {
    return () => {
      debouncedLabelUpdate.cancel();
    };
  }, [debouncedLabelUpdate]);

  const colorOptions = useMemo(
    () =>
      (Object.keys(GROUP_COLORS) as GroupColor[]).map((value) => ({
        value,
        dot: GROUP_COLORS[value].dot,
      })),
    [],
  );

  const handleDoubleClickLabel = useCallback(() => {
    setIsEditingLabel(true);
    requestAnimationFrame(() => inputRef.current?.focus());
  }, []);

  const commitLabel = useCallback(() => {
    setIsEditingLabel(false);
    const trimmed = localLabel.trim() || "Group";
    setLocalLabel(trimmed);
    debouncedLabelUpdate(trimmed);
    debouncedLabelUpdate.flush();
  }, [localLabel, debouncedLabelUpdate]);

  const handleLabelKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter" || e.key === "Escape") {
        e.preventDefault();
        commitLabel();
        inputRef.current?.blur();
      }
    },
    [commitLabel],
  );

  return (
    <div className="h-full w-full">
      <div
        className={cn(
          "group relative flex h-full w-full flex-col rounded-lg border-2 border-dashed",
          colorStyles.border,
          colorStyles.background,
          selected && "rounded-md ring-[3px] ring-sky-300 ring-offset-4",
        )}
      >
        <div className="canvas-node-drag-handle flex h-7 shrink-0 cursor-grab items-center rounded-t-md px-2">
          {isEditingLabel ? (
            <input
              ref={inputRef}
              value={localLabel}
              onChange={(e) => setLocalLabel(e.target.value)}
              onBlur={commitLabel}
              onKeyDown={handleLabelKeyDown}
              className={cn("nodrag w-full bg-transparent text-xs font-semibold outline-none", colorStyles.text)}
            />
          ) : (
            <span
              className={cn("cursor-text select-none text-xs font-semibold", colorStyles.text)}
              onDoubleClick={handleDoubleClickLabel}
            >
              {localLabel}
            </span>
          )}
        </div>

        {!hideActionsButton && (
          <>
            <div className="absolute -top-12 right-0 z-10 h-12 w-44 opacity-0" />
            <div className="nodrag absolute -top-8 right-0 z-10 hidden items-center gap-2 group-hover:flex">
              <div className="group/swatch relative flex items-center px-0.5 py-0.5">
                <button
                  type="button"
                  onClick={(event: React.MouseEvent) => {
                    event.preventDefault();
                    event.stopPropagation();
                  }}
                  className={cn("h-4 w-4 rounded-full border transition", GROUP_COLORS[activeColor].dot)}
                  aria-label={`Current group color: ${GROUP_COLORS[activeColor].label}`}
                />
                <div className="absolute right-0 top-1/2 hidden -translate-y-1/2 items-center gap-2 pr-0.5 group-hover/swatch:flex">
                  {colorOptions.map((option) => (
                    <button
                      key={option.value}
                      type="button"
                      onClick={(event: React.MouseEvent) => {
                        event.preventDefault();
                        event.stopPropagation();
                        onGroupUpdate?.({ color: option.value });
                      }}
                      className={cn("h-4 w-4 rounded-full border transition", option.dot)}
                      aria-label={`Set group color to ${GROUP_COLORS[option.value].label}`}
                    />
                  ))}
                </div>
              </div>
              {onUngroup && (
                <button
                  type="button"
                  onClick={(event: React.MouseEvent) => {
                    event.preventDefault();
                    event.stopPropagation();
                    onUngroup();
                  }}
                  className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800"
                  aria-label="Ungroup nodes"
                >
                  <Ungroup size={16} />
                </button>
              )}
              {onDelete && (
                <button
                  type="button"
                  onClick={(event: React.MouseEvent) => {
                    event.preventDefault();
                    event.stopPropagation();
                    onDelete();
                  }}
                  className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800"
                  aria-label="Delete group"
                >
                  <Trash2 size={16} />
                </button>
              )}
            </div>
          </>
        )}
      </div>
    </div>
  );
};

export const GroupNode = React.memo(
  GroupNodeBase,
  (prev, next) =>
    prev.groupLabel === next.groupLabel &&
    prev.groupColor === next.groupColor &&
    prev.selected === next.selected &&
    prev.hideActionsButton === next.hideActionsButton,
);
