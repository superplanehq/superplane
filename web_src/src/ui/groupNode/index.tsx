import React, { useCallback, useEffect, useMemo, useState } from "react";
import debounce from "lodash.debounce";
import { Trash2, Ungroup } from "lucide-react";
import { cn } from "@/lib/utils";
import { normalizeGroupColor } from "./constants";
import type { GroupColor } from "./constants";

type GroupColorStyles = {
  label: string;
  bgTint: string;
  border: string;
  glow: string;
  labelBg: string;
  labelText: string;
  dot: string;
};

const GROUP_COLORS: Record<GroupColor, GroupColorStyles> = {
  purple: {
    label: "Purple",
    bgTint: "bg-purple-500/5",
    border: "border-purple-300/90",
    glow: "shadow-[inset_0_0_0_1px_rgba(168,85,247,0.14),0_0_10px_-4px_rgba(147,51,234,0.12)]",
    labelBg: "bg-purple-50",
    labelText: "text-purple-900",
    dot: "bg-purple-300 border-purple-500/70",
  },
  blue: {
    label: "Blue",
    bgTint: "bg-sky-500/5",
    border: "border-sky-300/90",
    glow: "shadow-[inset_0_0_0_1px_rgba(14,165,233,0.14),0_0_10px_-4px_rgba(2,132,199,0.12)]",
    labelBg: "bg-sky-50",
    labelText: "text-sky-900",
    dot: "bg-sky-300 border-sky-500/70",
  },
  green: {
    label: "Green",
    bgTint: "bg-emerald-500/5",
    border: "border-emerald-300/90",
    glow: "shadow-[inset_0_0_0_1px_rgba(52,211,153,0.14),0_0_10px_-4px_rgba(16,185,129,0.12)]",
    labelBg: "bg-emerald-50",
    labelText: "text-emerald-900",
    dot: "bg-emerald-300 border-emerald-500/70",
  },
  cyan: {
    label: "Cyan",
    bgTint: "bg-cyan-500/5",
    border: "border-cyan-300/90",
    glow: "shadow-[inset_0_0_0_1px_rgba(34,211,238,0.14),0_0_10px_-4px_rgba(6,182,212,0.12)]",
    labelBg: "bg-cyan-50",
    labelText: "text-cyan-900",
    dot: "bg-cyan-300 border-cyan-500/70",
  },
  orange: {
    label: "Orange",
    bgTint: "bg-orange-500/5",
    border: "border-orange-300/90",
    glow: "shadow-[inset_0_0_0_1px_rgba(251,146,60,0.14),0_0_10px_-4px_rgba(234,88,12,0.12)]",
    labelBg: "bg-orange-50",
    labelText: "text-orange-900",
    dot: "bg-orange-300 border-orange-500/70",
  },
  rose: {
    label: "Rose",
    bgTint: "bg-rose-500/5",
    border: "border-rose-300/90",
    glow: "shadow-[inset_0_0_0_1px_rgba(251,113,133,0.14),0_0_10px_-4px_rgba(225,29,72,0.12)]",
    labelBg: "bg-rose-50",
    labelText: "text-rose-900",
    dot: "bg-rose-300 border-rose-500/70",
  },
  amber: {
    label: "Amber",
    bgTint: "bg-amber-500/5",
    border: "border-amber-300/90",
    glow: "shadow-[inset_0_0_0_1px_rgba(251,191,36,0.14),0_0_10px_-4px_rgba(217,119,6,0.12)]",
    labelBg: "bg-amber-50",
    labelText: "text-amber-900",
    dot: "bg-amber-300 border-amber-500/70",
  },
};

const GROUP_COLOR_KEYS = Object.keys(GROUP_COLORS) as GroupColor[];

export interface GroupNodeProps {
  groupLabel?: string;
  groupDescription?: string;
  groupColor?: GroupColor;
  selected?: boolean;
  hideActionsButton?: boolean;
  onGroupUpdate?: (updates: { label?: string; description?: string; color?: GroupColor }) => void;
  onUngroup?: () => void;
  onDelete?: () => void;
}

function stopEvent(event: React.MouseEvent) {
  event.preventDefault();
  event.stopPropagation();
}

function GroupActionsToolbar({
  activeColor,
  colorOptions,
  onGroupUpdate,
  onUngroup,
  onDelete,
}: {
  activeColor: GroupColor;
  colorOptions: { value: GroupColor; dot: string }[];
  onGroupUpdate?: GroupNodeProps["onGroupUpdate"];
  onUngroup?: () => void;
  onDelete?: () => void;
}) {
  return (
    <>
      <div className="absolute -top-12 right-0 z-20 h-12 w-44 opacity-0" aria-hidden />
      <div className="nodrag absolute -top-8 right-0 z-30 hidden flex-nowrap items-center justify-start gap-2 rounded-md border border-slate-200/80 bg-white/95 px-1.5 py-1 shadow-md group-hover:flex">
        <div className="group/swatch flex shrink-0 items-center gap-2 px-0.5 py-0.5">
          <div className="hidden shrink-0 flex-nowrap items-center gap-2 group-hover/swatch:flex">
            {colorOptions.map((option) => (
              <button
                key={option.value}
                type="button"
                onClick={(event: React.MouseEvent) => {
                  stopEvent(event);
                  onGroupUpdate?.({ color: option.value });
                }}
                className={cn("h-4 w-4 shrink-0 rounded-full border-2 transition", option.dot)}
                aria-label={`Set group color to ${GROUP_COLORS[option.value].label}`}
              />
            ))}
          </div>
          <button
            type="button"
            onClick={stopEvent}
            className={cn("h-4 w-4 shrink-0 rounded-full border-2 transition", GROUP_COLORS[activeColor].dot)}
            aria-label={`Current group color: ${GROUP_COLORS[activeColor].label}`}
          />
        </div>
        {onUngroup && (
          <button
            type="button"
            onClick={(event: React.MouseEvent) => {
              stopEvent(event);
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
              stopEvent(event);
              onDelete();
            }}
            className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800"
            aria-label="Delete group and contained nodes"
          >
            <Trash2 size={16} />
          </button>
        )}
      </div>
    </>
  );
}

function useGroupTextEditing(
  groupLabel: string,
  groupDescription: string,
  onGroupUpdate: GroupNodeProps["onGroupUpdate"],
) {
  const inputRef = React.useRef<HTMLInputElement | null>(null);
  const descriptionRef = React.useRef<HTMLTextAreaElement | null>(null);

  const [isEditingLabel, setIsEditingLabel] = useState(false);
  const [localLabel, setLocalLabel] = useState(groupLabel);
  const [isEditingDescription, setIsEditingDescription] = useState(false);
  const [localDescription, setLocalDescription] = useState(groupDescription);

  useEffect(() => {
    setLocalLabel(groupLabel);
  }, [groupLabel]);

  useEffect(() => {
    if (!isEditingDescription) setLocalDescription(groupDescription);
  }, [groupDescription, isEditingDescription]);

  useEffect(() => {
    if (isEditingDescription) requestAnimationFrame(() => descriptionRef.current?.focus());
  }, [isEditingDescription]);

  const onGroupUpdateRef = React.useRef(onGroupUpdate);
  useEffect(() => {
    onGroupUpdateRef.current = onGroupUpdate;
  }, [onGroupUpdate]);

  const debouncedLabelUpdate = useMemo(
    () => debounce((v: string) => onGroupUpdateRef.current?.({ label: v }), 500),
    [],
  );
  const debouncedDescriptionUpdate = useMemo(
    () => debounce((v: string) => onGroupUpdateRef.current?.({ description: v }), 500),
    [],
  );

  useEffect(() => {
    return () => {
      debouncedLabelUpdate.cancel();
      debouncedDescriptionUpdate.cancel();
    };
  }, [debouncedLabelUpdate, debouncedDescriptionUpdate]);

  const skipBlurCommitRef = React.useRef(false);

  const handleDoubleClickLabel = useCallback(() => {
    setIsEditingLabel(true);
    skipBlurCommitRef.current = false;
    requestAnimationFrame(() => inputRef.current?.focus());
  }, []);

  const commitLabel = useCallback(() => {
    if (skipBlurCommitRef.current) {
      skipBlurCommitRef.current = false;
      return;
    }
    setIsEditingLabel(false);
    const trimmed = localLabel.trim() || "Group";
    setLocalLabel(trimmed);
    debouncedLabelUpdate(trimmed);
    debouncedLabelUpdate.flush();
  }, [localLabel, debouncedLabelUpdate]);

  const handleLabelKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter") {
        e.preventDefault();
        inputRef.current?.blur();
        if (onGroupUpdate) setIsEditingDescription(true);
        return;
      }
      if (e.key === "Escape") {
        e.preventDefault();
        skipBlurCommitRef.current = true;
        debouncedLabelUpdate.cancel();
        setLocalLabel(groupLabel);
        setIsEditingLabel(false);
        inputRef.current?.blur();
      }
    },
    [onGroupUpdate, debouncedLabelUpdate, groupLabel],
  );

  const commitDescription = useCallback(() => {
    setIsEditingDescription(false);
    const trimmed = localDescription.trim();
    setLocalDescription(trimmed);
    debouncedDescriptionUpdate(trimmed);
    debouncedDescriptionUpdate.flush();
  }, [localDescription, debouncedDescriptionUpdate]);

  const handleDescriptionKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
      if (e.key === "Escape") {
        e.preventDefault();
        setLocalDescription(groupDescription);
        setIsEditingDescription(false);
        descriptionRef.current?.blur();
      }
      if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        commitDescription();
        descriptionRef.current?.blur();
      }
    },
    [commitDescription, groupDescription],
  );

  return {
    inputRef,
    descriptionRef,
    isEditingLabel,
    localLabel,
    setLocalLabel,
    isEditingDescription,
    setIsEditingDescription,
    localDescription,
    setLocalDescription,
    hasDescriptionText: Boolean(localDescription.trim()),
    handleDoubleClickLabel,
    commitLabel,
    handleLabelKeyDown,
    commitDescription,
    handleDescriptionKeyDown,
  };
}

const GroupNodeBase: React.FC<GroupNodeProps> = ({
  groupLabel = "Group",
  groupDescription = "",
  groupColor,
  selected = false,
  hideActionsButton,
  onGroupUpdate,
  onUngroup,
  onDelete,
}) => {
  const activeColor = normalizeGroupColor(groupColor);
  const colorStyles = GROUP_COLORS[activeColor];

  const colorOptions = useMemo(() => GROUP_COLOR_KEYS.map((value) => ({ value, dot: GROUP_COLORS[value].dot })), []);

  const text = useGroupTextEditing(groupLabel, groupDescription, onGroupUpdate);

  return (
    <div className="h-full w-full">
      <div
        className={cn(
          "group relative flex h-full w-full flex-col rounded-lg border-2",
          colorStyles.border,
          colorStyles.bgTint,
          colorStyles.glow,
          selected && "ring-[3px] ring-sky-400/80 ring-offset-2 ring-offset-white",
        )}
      >
        <div className={cn("shrink-0 rounded-t-md", colorStyles.labelBg)}>
          <div
            className={cn(
              "canvas-node-drag-handle flex min-h-10 cursor-grab items-start justify-start px-3 pt-2.5 text-left",
              text.isEditingDescription || text.hasDescriptionText ? "pb-1" : "pb-2.5",
            )}
          >
            <div className="min-w-0 flex-1 text-left">
              {text.isEditingLabel ? (
                <input
                  ref={text.inputRef}
                  value={text.localLabel}
                  onChange={(e) => text.setLocalLabel(e.target.value)}
                  onBlur={text.commitLabel}
                  onKeyDown={text.handleLabelKeyDown}
                  title="Press Enter to add a description"
                  className={cn(
                    "nodrag w-full bg-transparent text-left text-base font-semibold leading-snug outline-none",
                    colorStyles.labelText,
                  )}
                />
              ) : (
                <span
                  className={cn(
                    "inline-block w-fit max-w-full cursor-text select-none text-left text-base font-semibold leading-snug tracking-tight",
                    colorStyles.labelText,
                  )}
                  onDoubleClick={text.handleDoubleClickLabel}
                >
                  {text.localLabel}
                </span>
              )}
            </div>
          </div>

          {text.isEditingDescription ? (
            <div className="nodrag w-full px-3 pb-2.5 pt-0 text-left">
              <textarea
                ref={text.descriptionRef}
                value={text.localDescription}
                onChange={(e) => text.setLocalDescription(e.target.value)}
                onBlur={text.commitDescription}
                onKeyDown={text.handleDescriptionKeyDown}
                rows={3}
                placeholder="Description (optional)"
                className="w-full min-h-[4.5rem] resize-none border-0 bg-transparent text-left text-sm leading-snug text-slate-600 outline-none ring-0 placeholder:text-slate-400/70"
              />
            </div>
          ) : text.hasDescriptionText ? (
            <div className="nodrag w-full px-3 pb-2.5 pt-0 text-left">
              <p
                className="cursor-text select-none whitespace-pre-wrap text-left text-sm leading-snug text-slate-600 line-clamp-6"
                onDoubleClick={() => onGroupUpdate && text.setIsEditingDescription(true)}
              >
                {text.localDescription}
              </p>
            </div>
          ) : null}
        </div>

        {!hideActionsButton && (
          <GroupActionsToolbar
            activeColor={activeColor}
            colorOptions={colorOptions}
            onGroupUpdate={onGroupUpdate}
            onUngroup={onUngroup}
            onDelete={onDelete}
          />
        )}
      </div>
    </div>
  );
};

export const GroupNode = React.memo(
  GroupNodeBase,
  (prev, next) =>
    prev.groupLabel === next.groupLabel &&
    prev.groupDescription === next.groupDescription &&
    prev.groupColor === next.groupColor &&
    prev.selected === next.selected &&
    prev.hideActionsButton === next.hideActionsButton &&
    !!prev.onGroupUpdate === !!next.onGroupUpdate &&
    !!prev.onUngroup === !!next.onUngroup &&
    !!prev.onDelete === !!next.onDelete,
);
