import { useMemo, useState, useCallback } from "react";

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
}

/**
 * Simpler approach: we store the textarea value as plain text.
 * Mentions are tracked as "@Label" substrings that map to IDs.
 * On send, we replace each "@Label" with "[Label](node:id)" or "[Label](run:id)".
 */
export interface UseMentionsReturn {
  /** The textarea value (plain text with @Label for mentions) */
  value: string;
  /** Set the textarea value */
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
  /** Insert a mention at the current trigger position */
  insertMention: (item: MentionItem) => void;
  /** Get the serialized markdown output for sending */
  getMarkdown: () => string;
  /** All tracked mentions */
  mentions: InsertedMention[];
  /** Check if text is empty (ignoring whitespace) */
  isEmpty: boolean;
  /** Clear all content */
  clear: () => void;
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

export function useMentions(): UseMentionsReturn {
  const [value, setValue] = useState("");
  const [cursorPos, setCursorPos] = useState(0);
  const [mentions, setMentions] = useState<InsertedMention[]>([]);

  const trigger = useMemo(() => detectTrigger(value, cursorPos), [value, cursorPos]);

  const insertMention = useCallback(
    (item: MentionItem) => {
      // Replace from trigger start (@...) to cursor with "@Label "
      const displayText = `@${item.label}`;
      const before = value.slice(0, trigger.start);
      const after = value.slice(cursorPos);
      const newValue = before + displayText + " " + after;
      setValue(newValue);
      setCursorPos(before.length + displayText.length + 1);

      // Track this mention
      setMentions((prev) => [...prev, { type: item.type, id: item.id, label: item.label, displayText }]);
    },
    [value, cursorPos, trigger.start],
  );

  const getMarkdown = useCallback(() => {
    let result = value;
    // Replace each tracked mention's display text with markdown link
    // Process longer labels first to avoid partial matches
    const sorted = [...mentions].sort((a, b) => b.label.length - a.label.length);
    for (const m of sorted) {
      const displayText = `@${m.label}`;
      if (m.type === "run") {
        result = result.replaceAll(displayText, `[${m.label}](run:${m.id})`);
      } else {
        result = result.replaceAll(displayText, `[${m.label}](node:${m.id})`);
      }
    }
    return result;
  }, [value, mentions]);

  const isEmpty = value.trim().length === 0;

  const clear = useCallback(() => {
    setValue("");
    setCursorPos(0);
    setMentions([]);
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
  };
}
