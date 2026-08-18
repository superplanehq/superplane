export interface WorkOrderMentionCandidate {
  id: string;
  name: string;
  email?: string;
}

export interface WorkOrderMentionQuery {
  start: number;
  query: string;
}

const MAX_MENTION_SUGGESTIONS = 8;

export function mentionQueryAtCursor(value: string, cursor: number): WorkOrderMentionQuery | null {
  const safeCursor = Math.max(0, Math.min(cursor, value.length));
  const before = value.slice(0, safeCursor);
  const at = before.lastIndexOf("@");
  if (at < 0 || !isMentionTrigger(value, at)) {
    return null;
  }

  const query = before.slice(at + 1);
  if (/\s/.test(query)) {
    return null;
  }

  return { start: at, query };
}

export function filterMentionCandidates(
  candidates: WorkOrderMentionCandidate[],
  query: string,
): WorkOrderMentionCandidate[] {
  const needle = query.trim().toLowerCase();
  const matches = needle ? candidates.filter((candidate) => mentionCandidateMatches(candidate, needle)) : candidates;

  return matches.slice(0, MAX_MENTION_SUGGESTIONS);
}

export function insertMentionAtCursor(
  value: string,
  cursor: number,
  displayName: string,
): { value: string; cursor: number } {
  const inserted = `@${displayName} `;
  const mention = mentionQueryAtCursor(value, cursor);
  if (!mention) {
    return {
      value: value.slice(0, cursor) + inserted + value.slice(cursor),
      cursor: cursor + inserted.length,
    };
  }

  return {
    value: value.slice(0, mention.start) + inserted + value.slice(cursor),
    cursor: mention.start + inserted.length,
  };
}

export function retainMentions(mentions: WorkOrderMentionCandidate[], body: string): WorkOrderMentionCandidate[] {
  return mentions.filter((mention) => bodyContainsMentionToken(body, mention, mentions));
}

function bodyContainsMentionToken(
  body: string,
  mention: WorkOrderMentionCandidate,
  tracked: WorkOrderMentionCandidate[],
): boolean {
  const token = `@${mention.name}`;
  let from = 0;
  while (from <= body.length) {
    const index = body.indexOf(token, from);
    if (index < 0) {
      return false;
    }
    if (isCompleteMentionToken(body, index, token.length, mention, tracked)) {
      return true;
    }
    from = index + 1;
  }
  return false;
}

function isCompleteMentionToken(
  body: string,
  start: number,
  tokenLength: number,
  mention: WorkOrderMentionCandidate,
  tracked: WorkOrderMentionCandidate[],
): boolean {
  if (!isMentionTrigger(body, start)) {
    return false;
  }

  const end = start + tokenLength;
  if (end < body.length && /[A-Za-z0-9_]/.test(body[end] ?? "")) {
    return false;
  }

  return !tracked.some(
    (other) =>
      other.name.length > mention.name.length &&
      other.name.startsWith(mention.name) &&
      body.startsWith(`@${other.name}`, start),
  );
}

function isMentionTrigger(value: string, atIndex: number): boolean {
  if (value[atIndex] !== "@") {
    return false;
  }
  if (atIndex === 0) {
    return true;
  }

  const previous = value[atIndex - 1];
  return /\s/.test(previous) || previous === "(" || previous === "[" || previous === "{";
}

function mentionCandidateMatches(candidate: WorkOrderMentionCandidate, needle: string): boolean {
  return candidate.name.toLowerCase().includes(needle) || (candidate.email?.toLowerCase().includes(needle) ?? false);
}
