const SETUP_COMMAND = /^(?:rm|mkdir|echo|printf)\b|^git\s+config\b/;
const PROLOGUE_COMMAND = /^(?:set\s+-|export\s+[\w.]+=|cd(?:\s|$))/;

export function shellScriptHeadline(script: string): string {
  const lines = script.split(/\r?\n/);
  const candidates = commandCandidates(lines);
  const pool = candidates.length > 0 ? candidates : lines.map((line) => line.trim()).filter(Boolean);
  const chosen = pool.find((line) => !isSetupCommand(firstSubstantiveSegment(line))) ?? pool[0];
  return chosen ? firstSubstantiveSegment(chosen) : "";
}

export function shellScriptLineCount(script: string): number {
  return script.split(/\r?\n/).filter((line) => line.trim()).length;
}

export function normalizeTerminalOutput(text: string): string {
  if (!text) return "";
  const lines = stripAnsi(text)
    .split("\n")
    .map((line) => lastCarriageReturnSegment(line));
  return collapseBlankRuns(lines).join("\n");
}

function commandCandidates(lines: string[]): string[] {
  const candidates: string[] = [];
  for (let index = 0; index < lines.length; index += 1) {
    const line = lines[index];
    if (isPrologueLine(line)) continue;
    const guardEnd = guardBlockEnd(lines, index);
    if (guardEnd !== undefined) {
      index = guardEnd;
      continue;
    }
    const trimmed = line.trim();
    if (trimmed) candidates.push(trimmed);
  }
  return candidates;
}

function isPrologueLine(line: string): boolean {
  const trimmed = line.trim();
  if (!trimmed || trimmed.startsWith("#")) return true;
  if (hasChain(trimmed)) return false;
  return isPrologueSegment(trimmed);
}

function isPrologueSegment(segment: string): boolean {
  const trimmed = segment.trim();
  return !trimmed || trimmed.startsWith("#") || PROLOGUE_COMMAND.test(trimmed);
}

function isSetupCommand(segment: string): boolean {
  return SETUP_COMMAND.test(segment.trim());
}

function guardBlockEnd(lines: string[], start: number): number | undefined {
  if (!/^\s*if\b/.test(lines[start])) return undefined;
  let depth = 0;
  let exits = false;
  let runsCommand = false;
  for (let index = start; index < lines.length; index += 1) {
    const trimmed = lines[index].trim();
    if (/^if\b/.test(trimmed)) depth += 1;
    if (!isGuardBodyLine(trimmed)) runsCommand = true;
    if (/\bexit\b/.test(trimmed)) exits = true;
    if (/^fi\b/.test(trimmed)) {
      depth -= 1;
      if (depth === 0) return exits && !runsCommand ? index : undefined;
    }
  }
  return undefined;
}

function isGuardBodyLine(line: string): boolean {
  if (!line || line.startsWith("#")) return true;
  if (/^(?:if|then|else|elif|fi)\b/.test(line)) return true;
  if (/^(?:echo|printf)\b/.test(line)) return true;
  return /\bexit\b/.test(line);
}

function firstSubstantiveSegment(line: string): string {
  const segments = splitShellSegments(line);
  return segments.find((segment) => !isPrologueSegment(segment)) ?? segments[0] ?? line.trim();
}

function splitShellSegments(line: string): string[] {
  const segments: string[] = [];
  let current = "";
  let quote: "'" | '"' | undefined;
  for (let index = 0; index < line.length; index += 1) {
    const char = line[index];
    if (quote) {
      current += char;
      if (char === quote && !(quote === '"' && line[index - 1] === "\\")) quote = undefined;
      continue;
    }
    if (char === "'" || char === '"') {
      quote = char;
      current += char;
      continue;
    }
    if (char === ";" || (char === "&" && line[index + 1] === "&")) {
      pushSegment(segments, current);
      current = "";
      if (char === "&") index += 1;
      continue;
    }
    current += char;
  }
  pushSegment(segments, current);
  return segments;
}

function pushSegment(segments: string[], value: string): void {
  const trimmed = value.trim();
  if (trimmed) segments.push(trimmed);
}

function hasChain(line: string): boolean {
  return splitShellSegments(line).length > 1;
}

function lastCarriageReturnSegment(line: string): string {
  const parts = line.split("\r");
  for (let index = parts.length - 1; index >= 0; index -= 1) {
    if (parts[index].length > 0) return parts[index];
  }
  return "";
}

const ESCAPE = String.fromCharCode(0x1b);
const BELL = String.fromCharCode(0x07);

function stripAnsi(text: string): string {
  let result = "";
  for (let index = 0; index < text.length; index += 1) {
    if (text[index] !== ESCAPE) {
      result += text[index];
      continue;
    }
    index = skipAnsiSequence(text, index);
  }
  return result;
}

function skipAnsiSequence(text: string, index: number): number {
  const marker = text[index + 1];
  if (marker === "[") return skipCsi(text, index + 2);
  if (marker === "]") return skipOsc(text, index + 2);
  return index + 1;
}

function skipCsi(text: string, index: number): number {
  let cursor = index;
  while (cursor < text.length) {
    const code = text.charCodeAt(cursor);
    cursor += 1;
    if (code >= 0x40 && code <= 0x7e) return cursor - 1;
  }
  return text.length - 1;
}

function skipOsc(text: string, index: number): number {
  for (let cursor = index; cursor < text.length; cursor += 1) {
    if (text[cursor] === BELL) return cursor;
    if (text[cursor] === ESCAPE && text[cursor + 1] === "\\") return cursor + 1;
  }
  return text.length - 1;
}

function collapseBlankRuns(lines: string[]): string[] {
  const collapsed: string[] = [];
  let blankRun = 0;
  for (const line of lines) {
    if (!line.trim()) {
      blankRun += 1;
      continue;
    }
    if (blankRun > 0 && collapsed.length > 0) collapsed.push("");
    blankRun = 0;
    collapsed.push(line.trimEnd());
  }
  return collapsed;
}
