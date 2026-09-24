import { useMemo, useState, useCallback, useRef } from "react";

export type MentionType = "node" | "run";
export type ComposerTriggerKind = "mention" | "skill";

export interface MentionItem {
  type: MentionType;
  id: string;
  label: string;
  /** For nodes: component name. For runs: status */
  meta?: string;
}

export interface InsertedMention {
  type: MentionType;
  id: string;
  label: string;
  /** The display text in the textarea (e.g. "@PR Opened") */
  displayText: string;
  /** Character index where this mention starts in the value */
  startIndex: number;
}

/**
 * Mentions are tracked as "@Label" substrings with position info.
 * On send, we replace each tracked mention with "[Label](node:id)" or "[Label](run:id)".
 * Skill slash tags stay as `/command` in the sent text.
 */
export interface UseMentionsReturn {
  /** The textarea value (plain text with @Label for mentions) */
  value: string;
  /** Set the textarea value (also prunes stale mentions) */
  setValue: (v: string) => void;
  /** Whether the mention dropdown should be shown */
  showDropdown: boolean;
  /** Whether the skill slash dropdown should be shown */
  showSkillDropdown: boolean;
  /** Current filter text for the dropdown */
  filter: string;
  /** Position (character index) where the @ or / trigger starts */
  triggerStart: number;
  /** Current cursor position */
  cursorPos: number;
  /** Update cursor position */
  setCursorPos: (pos: number) => void;
  /** Insert a mention at the current trigger position. Returns the new cursor position. */
  insertMention: (item: MentionItem) => number;
  /** Insert a `/command` at the current slash trigger. Returns the new cursor position. */
  insertSkill: (command: string) => number;
  /** Get the serialized markdown output for sending */
  getMarkdown: () => string;
  /** All tracked mentions */
  mentions: InsertedMention[];
  /** Check if text is empty (ignoring whitespace) */
  isEmpty: boolean;
  /** Clear all content */
  clear: () => void;
  /** Store pre-send state for recovery */
  snapshot: () => void;
  /** Restore from snapshot (for failed sends) */
  restore: () => void;
  /** Dismiss the dropdown (Escape key) */
  dismiss: () => void;
}

/**
 * Detect if cursor is in a mention or skill trigger position.
 * Mention filters allow spaces so multi-word names can be matched.
 * Skill filters end at whitespace so `/command ` is not an active query.
 * Terminates on newline.
 */
function detectComposerTrigger(
  text: string,
  cursorPos: number,
): { kind: ComposerTriggerKind | null; filter: string; start: number } {
  const before = text.slice(0, cursorPos);

  for (let i = before.length - 1; i >= 0; i--) {
    const ch = before[i];
    if (ch === "\n") {
      return { kind: null, filter: "", start: 0 };
    }
    if (ch === "@" || ch === "/") {
      if (i === 0 || /\s/.test(before[i - 1])) {
        const filter = before.slice(i + 1);
        if (ch === "/" && /\s/.test(filter)) {
          return { kind: null, filter: "", start: 0 };
        }
        return { kind: ch === "@" ? "mention" : "skill", filter, start: i };
      }
      return { kind: null, filter: "", start: 0 };
    }
  }
  return { kind: null, filter: "", start: 0 };
}

/** Prune mentions whose @Label text no longer exists at the tracked position */
function pruneMentions(text: string, mentions: InsertedMention[]): InsertedMention[] {
  return mentions.filter((m) => {
    const expected = `@${m.label}`;
    return text.slice(m.startIndex, m.startIndex + expected.length) === expected;
  });
}

export function useMentions(): UseMentionsReturn {
  const [value, setRawValue] = useState("");
  const [cursorPos, setCursorPos] = useState(0);
  const [mentions, setMentions] = useState<InsertedMention[]>([]);
  const [dismissed, setDismissed] = useState(false);
  const snapshotRef = useRef<{ value: string; mentions: InsertedMention[] } | null>(null);

  const trigger = useMemo(() => detectComposerTrigger(value, cursorPos), [value, cursorPos]);

  // Check if the trigger position is inside an already-inserted mention
  const triggerIsInsertedMention = useMemo(() => {
    if (trigger.kind !== "mention") return false;
    return mentions.some((m) => {
      const expected = `@${m.label}`;
      return m.startIndex === trigger.start && value.slice(m.startIndex, m.startIndex + expected.length) === expected;
    });
  }, [trigger.kind, trigger.start, mentions, value]);

  const showDropdown = useMemo(() => {
    if (trigger.kind !== "mention") return false;
    if (dismissed) return false;
    if (triggerIsInsertedMention) return false;
    return true;
  }, [trigger.kind, dismissed, triggerIsInsertedMention]);

  const showSkillDropdown = useMemo(() => {
    if (trigger.kind !== "skill") return false;
    if (dismissed) return false;
    return true;
  }, [trigger.kind, dismissed]);

  // setValue that also prunes stale mentions
  const setValue = useCallback((v: string) => {
    setRawValue(v);
    setMentions((prev) => pruneMentions(v, prev));
    setDismissed(false); // Reset dismiss on any text change
  }, []);

  const setCursorPosWrapped = useCallback((pos: number) => {
    setCursorPos(pos);
  }, []);

  const dismiss = useCallback(() => {
    setDismissed(true);
  }, []);

  const insertMention = useCallback(
    (item: MentionItem): number => {
      const displayText = `@${item.label}`;
      const before = value.slice(0, trigger.start);
      const after = value.slice(cursorPos);
      const newValue = before + displayText + " " + after;
      const newCursorPos = before.length + displayText.length + 1;

      setRawValue(newValue);
      setCursorPos(newCursorPos);
      setDismissed(false);

      const insertionPoint = before.length;
      const delta = displayText.length + 1 - (cursorPos - trigger.start);
      const newMention: InsertedMention = {
        type: item.type,
        id: item.id,
        label: item.label,
        displayText,
        startIndex: insertionPoint,
      };
      setMentions((prev) => [
        ...prev.map((m) => (m.startIndex >= insertionPoint ? { ...m, startIndex: m.startIndex + delta } : m)),
        newMention,
      ]);

      return newCursorPos;
    },
    [value, cursorPos, trigger.start],
  );

  const insertSkill = useCallback(
    (command: string): number => {
      const displayText = `/${command}`;
      const before = value.slice(0, trigger.start);
      const after = value.slice(cursorPos);
      const newValue = before + displayText + " " + after;
      const newCursorPos = before.length + displayText.length + 1;

      setRawValue(newValue);
      setCursorPos(newCursorPos);
      setDismissed(true);
      return newCursorPos;
    },
    [value, cursorPos, trigger.start],
  );

  const getMarkdown = useCallback(() => {
    const sorted = [...mentions].sort((a, b) => b.startIndex - a.startIndex);
    let result = value;
    for (const m of sorted) {
      const displayText = `@${m.label}`;
      const at = m.startIndex;
      if (result.slice(at, at + displayText.length) !== displayText) continue;
      const link = m.type === "run" ? `[${m.label}](run:${m.id})` : `[${m.label}](node:${m.id})`;
      result = result.slice(0, at) + link + result.slice(at + displayText.length);
    }
    return result;
  }, [value, mentions]);

  const isEmpty = value.trim().length === 0;

  const snapshot = useCallback(() => {
    snapshotRef.current = { value, mentions: [...mentions] };
  }, [value, mentions]);

  const restore = useCallback(() => {
    if (snapshotRef.current) {
      setRawValue(snapshotRef.current.value);
      setCursorPos(snapshotRef.current.value.length);
      setMentions(snapshotRef.current.mentions);
      snapshotRef.current = null;
    }
  }, []);

  const clear = useCallback(() => {
    setRawValue("");
    setCursorPos(0);
    setMentions([]);
    setDismissed(false);
    // Don't null snapshotRef here — restore() needs it if send fails
  }, []);

  return {
    value,
    setValue,
    showDropdown,
    showSkillDropdown,
    filter: trigger.filter,
    triggerStart: trigger.start,
    cursorPos,
    setCursorPos: setCursorPosWrapped,
    insertMention,
    insertSkill,
    getMarkdown,
    mentions,
    isEmpty,
    clear,
    snapshot,
    restore,
    dismiss,
  };
}
