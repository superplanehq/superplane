/**
 * CloudWatch live logs emit each flush chunk as its own event. Claude stream
 * text often arrives mid-word across chunks; reassemble those fragments without
 * merging unrelated command output lines.
 */
const SECTION_BREAK = /^(Thinking|-> \[|→ |✓ |✗ |Claude Code started|Rate limit notice|Retrying API)/;
/** Labels removed from the formatter; still filtered so older runs render cleanly. */
const HIDDEN_LABEL = /^(Claude|← tool result(?: \(empty\))?)$/;

export function coalesceLiveLogLines(lines: string[]): string[] {
  const out: string[] = [];

  for (const line of lines) {
    if (HIDDEN_LABEL.test(line)) {
      continue;
    }

    if (line.trim() === "") {
      appendParagraphBreak(out);
      continue;
    }

    if (SECTION_BREAK.test(line) || startsWithWhitespace(line)) {
      out.push(line);
      continue;
    }

    const last = out[out.length - 1];
    if (
      last !== undefined &&
      !SECTION_BREAK.test(last) &&
      !startsWithWhitespace(last) &&
      (last.endsWith("\n\n") || shouldJoinFragments(last, line))
    ) {
      out[out.length - 1] = last + line;
      continue;
    }

    out.push(line);
  }

  return out;
}

function startsWithWhitespace(line: string): boolean {
  return /^\s/.test(line);
}

function shouldJoinFragments(prev: string, cur: string): boolean {
  const prevEnd = prev[prev.length - 1];
  const curStart = cur[0];
  if (prevEnd === undefined || curStart === undefined) {
    return false;
  }

  // Markdown / punctuation continuations: "1.5 MB" + "** ("
  if (/\S/.test(prevEnd) && /[*_`"'([{]/.test(curStart)) {
    return true;
  }

  // Mid-word split: "... bi" + "ggest" (short unfinished last token)
  if (/\w/.test(prevEnd) && /[a-z0-9]/.test(curStart)) {
    return lastToken(prev).length <= 3;
  }

  return false;
}

function lastToken(text: string): string {
  const parts = text.trim().split(/\s+/);
  return parts[parts.length - 1] ?? "";
}

function appendParagraphBreak(out: string[]): void {
  const last = out[out.length - 1];
  if (last === undefined || SECTION_BREAK.test(last) || last.endsWith("\n\n")) {
    return;
  }
  out[out.length - 1] = last.replace(/\n+$/, "") + "\n\n";
}
