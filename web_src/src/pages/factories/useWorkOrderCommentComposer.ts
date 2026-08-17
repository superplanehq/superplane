import { useLayoutEffect, useMemo, useRef, useState } from "react";
import { getCaretCoordinates } from "./lib/caretPosition";
import { buildMentionToken, detectMentionTrigger, extractMentionedUserIds } from "./lib/mentionComposer";
import { useWorkOrderMentionOptions, type WorkOrderMentionOption } from "./lib/workOrderMentionOptions";

const MENTION_MENU_GAP_PX = 4;

interface UseWorkOrderCommentComposerArgs {
  organizationId: string;
  canComment: boolean;
  isSubmitting: boolean;
  onSubmit: (body: string, mentionedUserIds: string[]) => Promise<void>;
}

/**
 * All state/behavior behind `WorkOrderCommentComposer`'s `@` mention picker:
 * trigger detection, the filtered member list, keyboard navigation, token
 * insertion, and submit. Split out of the component so the JSX stays a plain
 * render of what this returns (see `mentionComposer.ts` for why mentions are
 * plain `@[Name](user:id)` text rather than a hidden-markup overlay).
 */
export function useWorkOrderCommentComposer({
  organizationId,
  canComment,
  isSubmitting,
  onSubmit,
}: UseWorkOrderCommentComposerArgs) {
  const [body, setBody] = useState("");
  const [cursorPos, setCursorPos] = useState(0);
  const [dismissedTriggerStart, setDismissedTriggerStart] = useState<number | null>(null);
  const [highlightedIndex, setHighlightedIndex] = useState(0);
  const textareaRef = useRef<HTMLTextAreaElement | null>(null);
  // Selecting a mention replaces a range of the (controlled) textarea value,
  // which resets the DOM's own selection; this carries the caret position we
  // want restored to the layout effect below. A plain ref (not state) so the
  // effect can run — and clear it — every commit without retriggering itself.
  const pendingSelectionRef = useRef<number | null>(null);

  const canSubmit = canComment && Boolean(body.trim()) && !isSubmitting;

  const trigger = useMemo(() => detectMentionTrigger(body, cursorPos), [body, cursorPos]);
  const menuOpen = trigger.active && trigger.start !== dismissedTriggerStart;
  const mentionOptions = useWorkOrderMentionOptions(organizationId, menuOpen ? trigger.query : "");

  const [menuPosition, setMenuPosition] = useState<{ top: number; left: number } | null>(null);

  useLayoutEffect(() => {
    setHighlightedIndex(0);
  }, [trigger.start, trigger.query]);

  useLayoutEffect(() => {
    if (!menuOpen || !textareaRef.current) {
      setMenuPosition(null);
      return;
    }

    const caret = getCaretCoordinates(textareaRef.current, trigger.start + 1);
    setMenuPosition({ top: caret.top + caret.height + MENTION_MENU_GAP_PX, left: caret.left });
  }, [menuOpen, trigger.start]);

  // Runs after every commit (including the one that just replaced `body`),
  // synchronously before the browser can paint or process the next event —
  // unlike `requestAnimationFrame`, this can't race a fast follow-up
  // keystroke/selection in the same textarea.
  useLayoutEffect(() => {
    if (pendingSelectionRef.current === null || !textareaRef.current) {
      return;
    }
    const pos = pendingSelectionRef.current;
    pendingSelectionRef.current = null;
    textareaRef.current.setSelectionRange(pos, pos);
  });

  const showMenu = menuOpen && mentionOptions.length > 0 && Boolean(menuPosition);

  const syncCursor = (target: HTMLTextAreaElement) => {
    setCursorPos(target.selectionStart ?? target.value.length);
  };

  const selectMentionOption = (option: WorkOrderMentionOption) => {
    const token = buildMentionToken(option.name, option.id);
    const before = body.slice(0, trigger.start);
    const after = body.slice(cursorPos);
    const insertion = `${token} `;
    const nextBody = `${before}${insertion}${after}`;
    const nextCursorPos = before.length + insertion.length;

    pendingSelectionRef.current = nextCursorPos;
    setBody(nextBody);
    setCursorPos(nextCursorPos);
    setDismissedTriggerStart(null);
  };

  const handleSubmit = async () => {
    const trimmed = body.trim();
    if (!trimmed) return;

    try {
      await onSubmit(trimmed, extractMentionedUserIds(trimmed));
      setBody("");
      setCursorPos(0);
      setDismissedTriggerStart(null);
    } catch {
      // Toast surfaced from the action hook.
    }
  };

  const handleChange = (event: React.ChangeEvent<HTMLTextAreaElement>) => {
    setBody(event.target.value);
    syncCursor(event.target);
    setDismissedTriggerStart(null);
  };

  const handleKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (showMenu) {
      if (event.key === "ArrowDown") {
        event.preventDefault();
        setHighlightedIndex((index) => (index + 1) % mentionOptions.length);
        return;
      }
      if (event.key === "ArrowUp") {
        event.preventDefault();
        setHighlightedIndex((index) => (index - 1 + mentionOptions.length) % mentionOptions.length);
        return;
      }
      if (event.key === "Enter" || event.key === "Tab") {
        event.preventDefault();
        selectMentionOption(mentionOptions[highlightedIndex]);
        return;
      }
      if (event.key === "Escape") {
        event.preventDefault();
        setDismissedTriggerStart(trigger.start);
        return;
      }
    }

    if ((event.metaKey || event.ctrlKey) && event.key === "Enter" && canSubmit) {
      event.preventDefault();
      void handleSubmit();
    }
  };

  return {
    textareaRef,
    body,
    canSubmit,
    showMenu,
    menuPosition,
    mentionOptions,
    highlightedIndex,
    setHighlightedIndex,
    selectMentionOption,
    handleChange,
    handleKeyDown,
    handleSubmit,
    syncCursor,
  };
}
