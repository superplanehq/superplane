/**
 * Plain-text `@` mention support for `WorkOrderCommentComposer`.
 *
 * Mentions are written straight into the textarea value as
 * `@[Display Name](user:<uuid>)` — the same "custom link scheme" convention
 * `node:`/`integration:` links already use in `Markdown.tsx` — rather than
 * kept out-of-band like the agent console's mention textarea. That keeps the
 * composer a plain, uncontrolled-looking `<textarea>` (no backdrop overlay to
 * hide the raw markup) at the cost of showing the token literally while
 * composing; `WorkOrderActivityTimeline` renders it as a chip once posted.
 */

const UUID_PATTERN = "[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}";
const MENTION_TOKEN_PATTERN = `\\(user:(${UUID_PATTERN})\\)`;

export interface MentionTrigger {
  /** Whether the cursor sits right after an open, untyped `@query`. */
  active: boolean;
  /** Text typed after the `@` so far (used to filter the member list). */
  query: string;
  /** Index of the `@` character itself, or -1 when inactive. */
  start: number;
}

const INACTIVE_TRIGGER: MentionTrigger = { active: false, query: "", start: -1 };

/**
 * Detect an in-progress `@` mention trigger ending at `cursorPos`, using the
 * same "last `@` at line-start or after whitespace" heuristic as the agent
 * console's mention picker (`useMentions.ts`). An already-inserted
 * `@[Name](user:id)` token always contains `[`/`]`/`(`/`)` between the `@`
 * and the cursor, so placing the caret inside one never reopens the picker.
 */
export function detectMentionTrigger(text: string, cursorPos: number): MentionTrigger {
  const before = text.slice(0, Math.max(0, cursorPos));

  for (let i = before.length - 1; i >= 0; i--) {
    const ch = before[i];
    if (ch === "\n") {
      return INACTIVE_TRIGGER;
    }

    if (ch === "@") {
      const precededByBoundary = i === 0 || /\s/.test(before[i - 1]);
      if (!precededByBoundary) {
        return INACTIVE_TRIGGER;
      }

      const query = before.slice(i + 1);
      if (/[[\]()]/.test(query)) {
        return INACTIVE_TRIGGER;
      }

      return { active: true, query, start: i };
    }
  }

  return INACTIVE_TRIGGER;
}

/** Build the mention token inserted when a member is picked from the menu. */
export function buildMentionToken(name: string, userId: string): string {
  // Display names shouldn't contain markdown link syntax, but strip it
  // defensively so a stray `]`/`)` can't break the token.
  const safeName = name.replace(/[[\]()]/g, "").trim() || userId;
  return `@[${safeName}](user:${userId})`;
}

/**
 * Re-scan `body` for `user:<uuid>` tokens still present, deduped and in
 * document order. Called just before submit rather than tracked as separate
 * state, so editing/deleting part of a mention token "downgrades" it to
 * plain text for free — no stale id sticks around once its markup breaks.
 */
export function extractMentionedUserIds(body: string): string[] {
  const pattern = new RegExp(MENTION_TOKEN_PATTERN, "g");
  const seen = new Set<string>();
  const ids: string[] = [];

  let match: RegExpExecArray | null;
  while ((match = pattern.exec(body)) !== null) {
    const id = match[1];
    const key = id.toLowerCase();
    if (!seen.has(key)) {
      seen.add(key);
      ids.push(id);
    }
  }

  return ids;
}
