import type { SuperplaneUsersUser } from "@/api-client";

import { getOrgUserDisplayFromUser } from "./orgUserDisplay";

export interface WorkOrderMentionCandidate {
  id: string;
  name: string;
  email?: string;
  avatarUrl?: string;
}

export interface WorkOrderMentionQuery {
  start: number;
  query: string;
}

export interface WorkOrderMentionSegment {
  text: string;
  mention: boolean;
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
  const remaining = mentionTokenCounts(body, mentions);
  const kept: WorkOrderMentionCandidate[] = [];
  for (const mention of mentions) {
    const count = remaining.get(mention.name) ?? 0;
    if (count < 1) {
      continue;
    }
    remaining.set(mention.name, count - 1);
    kept.push(mention);
  }
  return kept;
}

export function mentionsInBody(
  candidates: WorkOrderMentionCandidate[],
  body: string,
  preferred: WorkOrderMentionCandidate[] = [],
): WorkOrderMentionCandidate[] {
  const seen = new Set<string>();
  const ordered: WorkOrderMentionCandidate[] = [];
  for (const person of [...preferred, ...candidates]) {
    if (seen.has(person.id)) {
      continue;
    }
    seen.add(person.id);
    ordered.push(person);
  }
  return retainMentions(ordered, body);
}

export function uniqueMentionNames(names: string[]): string[] {
  const seen = new Set<string>();
  const unique: string[] = [];
  for (const name of names) {
    const trimmed = name.trim();
    if (!trimmed || seen.has(trimmed)) {
      continue;
    }
    seen.add(trimmed);
    unique.push(trimmed);
  }
  return unique;
}

export function mentionCandidatesFromOrgUsers(users: SuperplaneUsersUser[]): WorkOrderMentionCandidate[] {
  return users.flatMap((user) => {
    const display = getOrgUserDisplayFromUser(user);
    if (!display) {
      return [];
    }
    return [
      {
        id: display.id,
        name: display.name,
        email: user.metadata?.email?.trim() || undefined,
        avatarUrl: display.avatarUrl,
      },
    ];
  });
}

export function mentionCandidateByName(
  people: WorkOrderMentionCandidate[],
  token: string,
): WorkOrderMentionCandidate | undefined {
  const name = token.startsWith("@") ? token.slice(1) : token;
  return people.find((person) => person.name === name);
}

export function splitMentionSegments(body: string, names: string[]): WorkOrderMentionSegment[] {
  const mentionNames = uniqueMentionNames(names);
  if (!body) {
    return [];
  }
  if (mentionNames.length === 0) {
    return [{ text: body, mention: false }];
  }

  const tracked = mentionNames.map((name) => ({ id: name, name }));
  const segments: WorkOrderMentionSegment[] = [];
  let cursor = 0;

  while (cursor < body.length) {
    const mention = nextCompleteMention(body, cursor, mentionNames, tracked);
    if (!mention) {
      segments.push({ text: body.slice(cursor), mention: false });
      break;
    }
    if (mention.start > cursor) {
      segments.push({ text: body.slice(cursor, mention.start), mention: false });
    }
    segments.push({ text: body.slice(mention.start, mention.end), mention: true });
    cursor = mention.end;
  }

  return segments;
}

function nextCompleteMention(
  body: string,
  from: number,
  names: string[],
  tracked: WorkOrderMentionCandidate[],
): { start: number; end: number } | null {
  let searchFrom = from;
  while (searchFrom < body.length) {
    const at = body.indexOf("@", searchFrom);
    if (at < 0) {
      return null;
    }
    const mention = completeMentionAt(body, at, names, tracked);
    if (mention) {
      return mention;
    }
    searchFrom = at + 1;
  }
  return null;
}

function completeMentionAt(
  body: string,
  start: number,
  names: string[],
  tracked: WorkOrderMentionCandidate[],
): { start: number; end: number } | null {
  let best: { start: number; end: number } | null = null;
  for (const name of names) {
    const token = `@${name}`;
    if (!body.startsWith(token, start)) {
      continue;
    }
    if (!isCompleteMentionToken(body, start, token.length, name, tracked)) {
      continue;
    }
    if (!best || token.length > best.end - best.start) {
      best = { start, end: start + token.length };
    }
  }
  return best;
}

function mentionTokenCounts(body: string, tracked: WorkOrderMentionCandidate[]): Map<string, number> {
  const counts = new Map<string, number>();
  const names = [...new Set(tracked.map((mention) => mention.name))];
  for (const name of names) {
    counts.set(name, countCompleteMentionTokens(body, name, tracked));
  }
  return counts;
}

function countCompleteMentionTokens(body: string, name: string, tracked: WorkOrderMentionCandidate[]): number {
  const token = `@${name}`;
  let from = 0;
  let count = 0;
  while (from <= body.length) {
    const index = body.indexOf(token, from);
    if (index < 0) {
      return count;
    }
    if (isCompleteMentionToken(body, index, token.length, name, tracked)) {
      count += 1;
    }
    from = index + 1;
  }
  return count;
}

function isCompleteMentionToken(
  body: string,
  start: number,
  tokenLength: number,
  mentionName: string,
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
      other.name.length > mentionName.length &&
      other.name.startsWith(mentionName) &&
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
