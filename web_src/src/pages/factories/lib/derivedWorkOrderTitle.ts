export const MAX_DERIVED_WORK_ORDER_TITLE_LENGTH = 256;

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
  return line.trim();
}
