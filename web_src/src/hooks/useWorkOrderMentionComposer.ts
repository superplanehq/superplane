import type { SuperplaneUsersUser } from "@/api-client";
import { getOrgUserDisplayFromUser } from "@/lib/orgUserDisplay";
import {
  filterMentionCandidates,
  insertMentionAtCursor,
  mentionQueryAtCursor,
  retainMentions,
  type WorkOrderMentionCandidate,
} from "@/lib/workOrderMentions";
import { useMemo, useRef, useState, type KeyboardEvent, type RefObject } from "react";

interface UseWorkOrderMentionComposerResult {
  body: string;
  mentionedUserIds: string[];
  textareaRef: RefObject<HTMLTextAreaElement | null>;
  suggestions: WorkOrderMentionCandidate[];
  highlightIndex: number;
  setHighlightIndex: (index: number) => void;
  handleChange: (value: string, cursor: number) => void;
  handleSelectMention: (candidate: WorkOrderMentionCandidate) => void;
  handleMentionKeyDown: (event: KeyboardEvent<HTMLTextAreaElement>) => boolean;
  reset: () => void;
}

export function useWorkOrderMentionComposer(users: SuperplaneUsersUser[]): UseWorkOrderMentionComposerResult {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [body, setBody] = useState("");
  const [cursor, setCursor] = useState(0);
  const [mentions, setMentions] = useState<WorkOrderMentionCandidate[]>([]);
  const [highlightIndex, setHighlightIndex] = useState(0);
  const [dismissed, setDismissed] = useState(false);

  const candidates = useMemo(() => organizationMentionCandidates(users), [users]);
  const query = mentionQueryAtCursor(body, cursor);
  const suggestions = query && !dismissed ? filterMentionCandidates(candidates, query.query) : [];

  const applyBody = (nextBody: string, nextCursor: number, nextMentions?: WorkOrderMentionCandidate[]) => {
    const previousQuery = mentionQueryAtCursor(body, cursor);
    const nextQuery = mentionQueryAtCursor(nextBody, nextCursor);
    setBody(nextBody);
    setCursor(nextCursor);
    setMentions(retainMentions(nextMentions ?? mentions, nextBody));
    setHighlightIndex(0);

    const sameTrigger =
      nextBody === body && previousQuery !== null && nextQuery !== null && previousQuery.start === nextQuery.start;
    if (!sameTrigger) {
      setDismissed(false);
    }
  };

  const handleSelectMention = (candidate: WorkOrderMentionCandidate) => {
    const inserted = insertMentionAtCursor(body, cursor, candidate.name);
    const nextMentions = mentions.some((mention) => mention.id === candidate.id) ? mentions : [...mentions, candidate];
    applyBody(inserted.value, inserted.cursor, nextMentions);
    requestAnimationFrame(() => {
      const textarea = textareaRef.current;
      if (!textarea) return;
      textarea.focus();
      textarea.setSelectionRange(inserted.cursor, inserted.cursor);
    });
  };

  const handleMentionKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>): boolean => {
    return applyMentionMenuKeyDown(event, {
      suggestions,
      highlightIndex,
      onHighlight: setHighlightIndex,
      onSelect: handleSelectMention,
      onDismiss: () => setDismissed(true),
    });
  };

  return {
    body,
    mentionedUserIds: mentions.map((mention) => mention.id),
    textareaRef,
    suggestions,
    highlightIndex,
    setHighlightIndex,
    handleChange: (value, nextCursor) => applyBody(value, nextCursor),
    handleSelectMention,
    handleMentionKeyDown,
    reset: () => applyBody("", 0, []),
  };
}

function applyMentionMenuKeyDown(
  event: KeyboardEvent<HTMLTextAreaElement>,
  menu: {
    suggestions: WorkOrderMentionCandidate[];
    highlightIndex: number;
    onHighlight: (index: number) => void;
    onSelect: (candidate: WorkOrderMentionCandidate) => void;
    onDismiss: () => void;
  },
): boolean {
  if (menu.suggestions.length === 0) {
    return false;
  }
  if (event.key === "ArrowDown") {
    event.preventDefault();
    menu.onHighlight((menu.highlightIndex + 1) % menu.suggestions.length);
    return true;
  }
  if (event.key === "ArrowUp") {
    event.preventDefault();
    menu.onHighlight((menu.highlightIndex - 1 + menu.suggestions.length) % menu.suggestions.length);
    return true;
  }
  if (event.key === "Enter" || event.key === "Tab") {
    event.preventDefault();
    const selected = menu.suggestions[menu.highlightIndex] ?? menu.suggestions[0];
    if (selected) menu.onSelect(selected);
    return true;
  }
  if (event.key === "Escape") {
    event.preventDefault();
    menu.onDismiss();
    return true;
  }
  return false;
}

function organizationMentionCandidates(users: SuperplaneUsersUser[]): WorkOrderMentionCandidate[] {
  return users.flatMap((user) => {
    const display = getOrgUserDisplayFromUser(user);
    if (!display) {
      return [];
    }
    return [{ id: display.id, name: display.name, email: user.metadata?.email?.trim() || undefined }];
  });
}
