import React, { useCallback, useEffect, useMemo, useState } from "react";
import debounce from "lodash.debounce";
import { Trash2, Ungroup } from "lucide-react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { CONFIRM_DELETE_GROUP_MESSAGE, normalizeGroupColor } from "./constants";
import type { GroupColor } from "./constants";

type GroupColorStyles = {
  label: string;
  bgTint: string;
  border: string;
  labelBg: string;
  labelText: string;
  dot: string;
};

const GROUP_COLORS: Record<GroupColor, GroupColorStyles> = {
  purple: {
    label: "Purple",
    bgTint: "bg-purple-500/5",
    border: "border-purple-500/90",
    labelBg: "bg-transparent",
    labelText: "text-purple-600",
    dot: "bg-purple-200 border-purple-500",
  },
  blue: {
    label: "Blue",
    bgTint: "bg-sky-500/5",
    border: "border-sky-500/90",
    labelBg: "bg-transparent",
    labelText: "text-sky-600",
    dot: "bg-sky-200 border-sky-500",
  },
  green: {
    label: "Green",
    bgTint: "bg-green-500/5",
    border: "border-green-500/90",
    labelBg: "bg-transparent",
    labelText: "text-green-600",
    dot: "bg-green-200 border-green-500",
  },
  cyan: {
    label: "Cyan",
    bgTint: "bg-cyan-500/5",
    border: "border-cyan-500/90",
    labelBg: "bg-transparent",
    labelText: "text-cyan-600",
    dot: "bg-cyan-200 border-cyan-500",
  },
  orange: {
    label: "Orange",
    bgTint: "bg-orange-500/5",
    border: "border-orange-500/90",
    labelBg: "bg-transparent",
    labelText: "text-orange-600",
    dot: "bg-orange-200 border-orange-500",
  },
  rose: {
    label: "Rose",
    bgTint: "bg-rose-500/5",
    border: "border-rose-500/90",
    labelBg: "bg-transparent",
    labelText: "text-rose-600",
    dot: "bg-rose-200 border-rose-500",
  },
  amber: {
    label: "Amber",
    bgTint: "bg-amber-500/5",
    border: "border-amber-500/90",
    labelBg: "bg-transparent",
    labelText: "text-amber-600",
    dot: "bg-amber-200 border-amber-500",
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
      <div className="absolute -top-12 right-0 z-10 h-12 w-44 opacity-0" aria-hidden />
      <div className="absolute -top-8 right-0 z-10 hidden items-center gap-2 group-hover:flex nodrag">
        <div className="group/swatch relative flex items-center px-0.5 py-0.5">
          <button
            type="button"
            onClick={stopEvent}
            className={cn("h-4 w-4 rounded-full border transition", GROUP_COLORS[activeColor].dot)}
            aria-label={`Current group color: ${GROUP_COLORS[activeColor].label}`}
          />
          <div className="absolute right-0 top-1/2 hidden -translate-y-1/2 items-center gap-2 pr-0.5 group-hover/swatch:flex">
            {colorOptions.map((option) => (
              <button
                key={option.value}
                type="button"
                onClick={(event: React.MouseEvent) => {
                  stopEvent(event);
                  onGroupUpdate?.({ color: option.value });
                }}
                className={cn("h-4 w-4 rounded-full border transition", option.dot)}
                aria-label={`Set group color to ${GROUP_COLORS[option.value].label}`}
              />
            ))}
          </div>
        </div>
        {onUngroup && (
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                onClick={(event: React.MouseEvent) => {
                  stopEvent(event);
                  onUngroup();
                }}
                className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800"
                aria-label="Ungroup"
              >
                <Ungroup size={16} />
              </button>
            </TooltipTrigger>
            <TooltipContent>Ungroup</TooltipContent>
          </Tooltip>
        )}
        {onDelete && (
          <Tooltip>
            <TooltipTrigger asChild>
              <button
                type="button"
                onClick={(event: React.MouseEvent) => {
                  stopEvent(event);
                  if (!window.confirm(CONFIRM_DELETE_GROUP_MESSAGE)) {
                    return;
                  }
                  onDelete();
                }}
                className="flex items-center justify-center p-1 text-gray-500 transition hover:text-gray-800"
                aria-label="Delete Group"
              >
                <Trash2 size={16} />
              </button>
            </TooltipTrigger>
            <TooltipContent>Delete Group</TooltipContent>
          </Tooltip>
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
          "group relative flex h-full w-full flex-col rounded-lg border",
          colorStyles.border,
          colorStyles.bgTint,
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
