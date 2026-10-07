import type { SuperplaneUsersUser } from "@/api-client";
import {
  filterSkillSlashCandidates,
  insertSkillSlashAtCursor,
  skillSlashQueryAtCursor,
  type SkillSlashCandidate,
} from "@/lib/skillSlash";
import {
  filterMentionCandidates,
  insertMentionAtCursor,
  mentionCandidatesFromOrgUsers,
  mentionQueryAtCursor,
  mentionsInBody,
  type WorkOrderMentionCandidate,
} from "@/lib/workOrderMentions";
import { useMemo, useRef, useState, type KeyboardEvent, type RefObject } from "react";

interface UseWorkOrderMentionComposerResult {
  body: string;
  mentionedUserIds: string[];
  mentionPeople: WorkOrderMentionCandidate[];
  textareaRef: RefObject<HTMLTextAreaElement | null>;
  suggestions: WorkOrderMentionCandidate[];
  skillSuggestions: SkillSlashCandidate[];
  highlightIndex: number;
  setHighlightIndex: (index: number) => void;
  handleChange: (value: string, cursor: number) => void;
  handleSelectMention: (candidate: WorkOrderMentionCandidate) => void;
  handleSelectSkill: (candidate: SkillSlashCandidate) => void;
  handleMentionKeyDown: (event: KeyboardEvent<HTMLTextAreaElement>) => boolean;
  reset: () => void;
}

export function useWorkOrderMentionComposer(
  users: SuperplaneUsersUser[],
  skills: SkillSlashCandidate[] = [],
): UseWorkOrderMentionComposerResult {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const [body, setBody] = useState("");
  const [cursor, setCursor] = useState(0);
  const [mentions, setMentions] = useState<WorkOrderMentionCandidate[]>([]);
  const [highlightIndex, setHighlightIndex] = useState(0);
  const [dismissed, setDismissed] = useState(false);

  const candidates = useMemo(() => mentionCandidatesFromOrgUsers(users), [users]);
  const mentionQuery = mentionQueryAtCursor(body, cursor);
  const skillQuery = skillSlashQueryAtCursor(body, cursor);
  const skillActive = Boolean(skillQuery && (!mentionQuery || skillQuery.start >= mentionQuery.start));
  const mentionActive = Boolean(mentionQuery && !skillActive);
  const suggestions =
    mentionActive && mentionQuery && !dismissed ? filterMentionCandidates(candidates, mentionQuery.query) : [];
  const skillSuggestions =
    skillActive && skillQuery && !dismissed ? filterSkillSlashCandidates(skills, skillQuery.query) : [];

  const applyBody = (nextBody: string, nextCursor: number, nextMentions?: WorkOrderMentionCandidate[]) => {
    const previousMention = mentionQueryAtCursor(body, cursor);
    const nextMention = mentionQueryAtCursor(nextBody, nextCursor);
    const previousSkill = skillSlashQueryAtCursor(body, cursor);
    const nextSkill = skillSlashQueryAtCursor(nextBody, nextCursor);
    setBody(nextBody);
    setCursor(nextCursor);
    setMentions(mentionsInBody(candidates, nextBody, nextMentions ?? mentions));
    setHighlightIndex(0);

    const sameMention =
      nextBody === body &&
      previousMention !== null &&
      nextMention !== null &&
      previousMention.start === nextMention.start;
    const sameSkill =
      nextBody === body && previousSkill !== null && nextSkill !== null && previousSkill.start === nextSkill.start;
    if (!sameMention && !sameSkill) {
      setDismissed(false);
    }
  };

  const focusCursor = (nextCursor: number) => {
    requestAnimationFrame(() => {
      const textarea = textareaRef.current;
      if (!textarea) return;
      textarea.focus();
      textarea.setSelectionRange(nextCursor, nextCursor);
    });
  };

  const handleSelectMention = (candidate: WorkOrderMentionCandidate) => {
    const inserted = insertMentionAtCursor(body, cursor, candidate.name);
    const nextMentions = mentions.some((mention) => mention.id === candidate.id) ? mentions : [...mentions, candidate];
    applyBody(inserted.value, inserted.cursor, nextMentions);
    focusCursor(inserted.cursor);
  };

  const handleSelectSkill = (candidate: SkillSlashCandidate) => {
    const inserted = insertSkillSlashAtCursor(body, cursor, candidate.command);
    applyBody(inserted.value, inserted.cursor);
    focusCursor(inserted.cursor);
  };

  const handleMentionKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>): boolean => {
    if (skillSuggestions.length > 0) {
      return applyComposerMenuKeyDown(event, {
        suggestions: skillSuggestions,
        highlightIndex,
        onHighlight: setHighlightIndex,
        onSelect: handleSelectSkill,
        onDismiss: () => setDismissed(true),
      });
    }
    return applyComposerMenuKeyDown(event, {
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
    mentionPeople: candidates,
    textareaRef,
    suggestions,
    skillSuggestions,
    highlightIndex,
    setHighlightIndex,
    handleChange: (value, nextCursor) => applyBody(value, nextCursor),
    handleSelectMention,
    handleSelectSkill,
    handleMentionKeyDown,
    reset: () => applyBody("", 0, []),
  };
}

function applyComposerMenuKeyDown<T>(
  event: KeyboardEvent<HTMLTextAreaElement>,
  menu: {
    suggestions: T[];
    highlightIndex: number;
    onHighlight: (index: number) => void;
    onSelect: (candidate: T) => void;
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
  if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
    return false;
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
