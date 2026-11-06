// Escape untrusted values inserted into Markdown text and angle-bracket link destinations.
export function markdownText(value: unknown): string {
  if (typeof value !== "string") throw new Error("markdownText() expects a string");
  return Array.from(value, (character) => {
    const code = character.codePointAt(0)!;
    if (code <= 0x1f || (code >= 0x7f && code <= 0x9f) || code === 0x2028 || code === 0x2029) return " ";
    if ("&<>|\\`*_{}[]()!#~".includes(character)) return `&#${code};`;
    return character;
  }).join("");
}

export function markdownLink(value: unknown): string {
  if (typeof value !== "string") throw new Error("markdownLink() expects a string");
  return value.replace(/[<>\n\r ]/g, (character) => encodeURIComponent(character));
}
