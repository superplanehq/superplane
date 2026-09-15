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
  if (!isEmphasisDelimiterAt(line, openAt, delimiter)) {
    return -1;
  }
  const innerStart = openAt + delimiter.length;
  for (let index = innerStart; index < line.length; index += 1) {
    if (index === innerStart || !isEmphasisDelimiterAt(line, index, delimiter)) {
      continue;
    }
    return index;
  }
  return -1;
}

function isEmphasisDelimiterAt(line: string, index: number, delimiter: string): boolean {
  if (!line.startsWith(delimiter, index)) {
    return false;
  }
  if (delimiter !== "_") {
    return true;
  }
  const left = index > 0 ? line[index - 1] : "";
  const right = index + 1 < line.length ? line[index + 1] : "";
  return !isWordChar(left) || !isWordChar(right);
}

function isWordChar(char: string): boolean {
  return /^[A-Za-z0-9]$/.test(char);
}
