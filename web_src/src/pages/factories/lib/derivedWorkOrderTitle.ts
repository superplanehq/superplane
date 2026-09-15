export const MAX_DERIVED_WORK_ORDER_TITLE_LENGTH = 256;

const EMPHASIS_DELIMITERS = ["**", "__", "*", "_"] as const;

export function derivedWorkOrderTitle(markdown: string, maxLength = MAX_DERIVED_WORK_ORDER_TITLE_LENGTH): string {
  for (const raw of markdown.split(/\r?\n/)) {
    const line = unwrapTitleLine(raw);
    if (line) {
      return line.slice(0, maxLength);
    }
  }
  return "";
}

function unwrapTitleLine(raw: string): string {
  let line = raw.trim();
  if (!line || /^!\[/.test(line)) {
    return "";
  }
  line = line.replace(/^#{1,6}\s+/, "");
  const link = /^\[([^\]]+)\]\([^)]+\)$/.exec(line);
  if (link?.[1]) {
    line = link[1];
  }
  return unwrapPairedEmphasis(line);
}

function unwrapPairedEmphasis(line: string): string {
  let current = line;
  let previous = "";
  while (current !== previous) {
    previous = current;
    for (const delimiter of EMPHASIS_DELIMITERS) {
      current = unwrapDelimiter(current, delimiter);
    }
  }
  return current.replace(/\s+/g, " ").trim();
}

function unwrapDelimiter(line: string, delimiter: string): string {
  let result = "";
  let index = 0;
  while (index < line.length) {
    const codeEnd = closedInlineCodeSpanEnd(line, index);
    if (codeEnd !== -1) {
      result += line.slice(index, codeEnd);
      index = codeEnd;
      continue;
    }
    const closeAt = findEmphasisClose(line, index, delimiter);
    if (closeAt === -1) {
      result += line[index];
      index += 1;
      continue;
    }
    result += line.slice(index + delimiter.length, closeAt);
    index = closeAt + delimiter.length;
  }
  return result;
}

function findEmphasisClose(line: string, openAt: number, delimiter: string): number {
  if (!canOpenEmphasisAt(line, openAt, delimiter)) {
    return -1;
  }
  const innerStart = openAt + delimiter.length;
  let index = innerStart;
  while (index < line.length) {
    const codeEnd = closedInlineCodeSpanEnd(line, index);
    if (codeEnd !== -1) {
      index = codeEnd;
      continue;
    }
    if (index > innerStart && canCloseEmphasisAt(line, index, delimiter)) {
      return index;
    }
    index += 1;
  }
  return -1;
}

function canOpenEmphasisAt(line: string, index: number, delimiter: string): boolean {
  if (!isEmphasisDelimiterAt(line, index, delimiter)) {
    return false;
  }
  const after = line[index + delimiter.length] ?? "";
  return after !== "" && !isWhitespace(after);
}

function canCloseEmphasisAt(line: string, index: number, delimiter: string): boolean {
  if (!isEmphasisDelimiterAt(line, index, delimiter)) {
    return false;
  }
  const before = line[index - 1] ?? "";
  return before !== "" && !isWhitespace(before);
}

function isEmphasisDelimiterAt(line: string, index: number, delimiter: string): boolean {
  if (isEscapedAt(line, index) || !line.startsWith(delimiter, index)) {
    return false;
  }
  if (delimiter !== "_" && delimiter !== "__") {
    return true;
  }
  return isUnderscoreRunStart(line, index) && !isIntrawordUnderscoreRun(line, index);
}

function isUnderscoreRunStart(line: string, index: number): boolean {
  return index === 0 || line[index - 1] !== "_" || isEscapedAt(line, index - 1);
}

function isIntrawordUnderscoreRun(line: string, index: number): boolean {
  const left = index > 0 ? line[index - 1] : "";
  let runEnd = index;
  while (runEnd < line.length && line[runEnd] === "_") {
    runEnd += 1;
  }
  const right = runEnd < line.length ? line[runEnd] : "";
  return isWordChar(left) && isWordChar(right);
}

function closedInlineCodeSpanEnd(line: string, start: number): number {
  if (line[start] !== "`" || isEscapedAt(line, start)) {
    return -1;
  }
  let openerLength = 0;
  while (start + openerLength < line.length && line[start + openerLength] === "`") {
    openerLength += 1;
  }
  let index = start + openerLength;
  while (index < line.length) {
    if (line[index] !== "`") {
      index += 1;
      continue;
    }
    let closerLength = 0;
    while (index + closerLength < line.length && line[index + closerLength] === "`") {
      closerLength += 1;
    }
    if (closerLength === openerLength) {
      return index + closerLength;
    }
    index += closerLength;
  }
  return -1;
}

function isEscapedAt(line: string, index: number): boolean {
  let slashes = 0;
  for (let cursor = index - 1; cursor >= 0 && line[cursor] === "\\"; cursor -= 1) {
    slashes += 1;
  }
  return slashes % 2 === 1;
}

function isWhitespace(char: string): boolean {
  return /\s/.test(char);
}

function isWordChar(char: string): boolean {
  return /^[A-Za-z0-9]$/.test(char);
}
