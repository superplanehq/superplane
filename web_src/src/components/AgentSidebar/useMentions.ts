import { useMemo, useState, useCallback, useRef } from "react";

export type MentionType = "node" | "run";

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
 */
export interface UseMentionsReturn {
  /** The textarea value (plain text with @Label for mentions) */
  value: string;
  /** Set the textarea value (also prunes stale mentions) */
  setValue: (v: string) => void;
  /** Whether the mention dropdown should be shown */
  showDropdown: boolean;
  /** Current filter text for the dropdown */
  filter: string;
  /** Position (character index) where the @ trigger starts */
  triggerStart: number;
  /** Current cursor position */
  cursorPos: number;
  /** Update cursor position */
  setCursorPos: (pos: number) => void;
  /** Insert a mention at the current trigger position. Returns the new cursor position. */
  insertMention: (item: MentionItem) => number;
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
}

/** Detect if cursor is in a mention trigger position (after @) */
function detectTrigger(text: string, cursorPos: number): { active: boolean; filter: string; start: number } {
  const before = text.slice(0, cursorPos);

  // Find the last @ that's either at start or after whitespace
  for (let i = before.length - 1; i >= 0; i--) {
    const ch = before[i];
    // If we hit whitespace or newline without finding @, no trigger
    if (ch === " " || ch === "\n" || ch === "\t") {
      return { active: false, filter: "", start: 0 };
    }
    if (ch === "@") {
      // Check it's at start or after whitespace
      if (i === 0 || /\s/.test(before[i - 1])) {
        const filter = before.slice(i + 1);
        return { active: true, filter, start: i };
      }
      return { active: false, filter: "", start: 0 };
    }
  }
  return { active: false, filter: "", start: 0 };
}

/** Prune mentions whose @Label text no longer exists at the tracked position */
function pruneMentions(text: string, mentions: InsertedMention[]): InsertedMention[] {
  return mentions.filter((m) => {
    const expected = `@${m.label}`;
    // Check if the mention still exists at its tracked position
    if (text.slice(m.startIndex, m.startIndex + expected.length) === expected) {
      return true;
    }
    // Mention was edited/deleted — remove it
    return false;
  });
}

export function useMentions(): UseMentionsReturn {
  const [value, setValue_] = useState("");
  const [cursorPos, setCursorPos] = useState(0);
  const [mentions, setMentions] = useState<InsertedMention[]>([]);
  const snapshotRef = useRef<{ value: string; mentions: InsertedMention[] } | null>(null);

  const trigger = useMemo(() => detectTrigger(value, cursorPos), [value, cursorPos]);

  // setValue that also prunes stale mentions
  const setValue = useCallback((v: string) => {
    setValue_(v);
    setMentions((prev) => pruneMentions(v, prev));
  }, []);

  const insertMention = useCallback(
    (item: MentionItem): number => {
      // Replace from trigger start (@...) to cursor with "@Label "
      const displayText = `@${item.label}`;
      const before = value.slice(0, trigger.start);
      const after = value.slice(cursorPos);
      const newValue = before + displayText + " " + after;
      const newCursorPos = before.length + displayText.length + 1;

      setValue_(newValue);
      setCursorPos(newCursorPos);

      // Track this mention with position
      const insertionPoint = before.length;
      const delta = displayText.length + 1 - (cursorPos - trigger.start); // +1 for trailing space
      const newMention: InsertedMention = {
        type: item.type,
        id: item.id,
        label: item.label,
        displayText,
        startIndex: insertionPoint,
      };
      // Shift existing mentions that come after the insertion point
      setMentions((prev) => [
        ...prev.map((m) => (m.startIndex >= insertionPoint ? { ...m, startIndex: m.startIndex + delta } : m)),
        newMention,
      ]);

      return newCursorPos;
    },
    [value, cursorPos, trigger.start],
  );

  const getMarkdown = useCallback(() => {
    // Replace mentions by position (from end to start to preserve indices)
    const sorted = [...mentions].sort((a, b) => b.startIndex - a.startIndex);
    let result = value;
    for (const m of sorted) {
      const displayText = `@${m.label}`;
      const at = m.startIndex;
      // Verify the mention is still at this position
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
      setValue_(snapshotRef.current.value);
      setMentions(snapshotRef.current.mentions);
      snapshotRef.current = null;
    }
  }, []);

  const clear = useCallback(() => {
    setValue_("");
    setCursorPos(0);
    setMentions([]);
    snapshotRef.current = null;
  }, []);

  return {
    value,
    setValue,
    showDropdown: trigger.active,
    filter: trigger.filter,
    triggerStart: trigger.start,
    cursorPos,
    setCursorPos,
    insertMention,
    getMarkdown,
    mentions,
    isEmpty,
    clear,
    snapshot,
    restore,
  };
}
