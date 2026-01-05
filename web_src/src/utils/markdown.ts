/**
 * Lightweight markdown parser for basic formatting.
 * Supports: bold, italic, headings (h1-h3), and line breaks.
 * XSS-safe: HTML is escaped before markdown processing.
 */
export function parseBasicMarkdown(text: string): string {
  if (!text) return "";

  // Escape HTML first to prevent XSS
  let html = text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");

  // Process headings (must be at start of line) - use special marker to preserve newlines after them
  html = html.replace(/^### (.+)$/gm, '<h3 class="text-base font-bold mt-3 mb-1">$1</h3>###HEADING###');
  html = html.replace(/^## (.+)$/gm, '<h2 class="text-lg font-bold mt-3 mb-1">$1</h2>###HEADING###');
  html = html.replace(/^# (.+)$/gm, '<h1 class="text-xl font-bold mt-4 mb-1">$1</h1>###HEADING###');

  // Process bold (**text**)
  html = html.replace(/\*\*(.+?)\*\*/g, '<strong class="font-bold">$1</strong>');

  // Process italic (*text*) - must come after bold
  html = html.replace(/\*(.+?)\*/g, '<em class="italic">$1</em>');

  // Convert single line breaks to <br> with minimal spacing
  html = html.replace(/\n/g, '<br class="leading-none">');

  // Remove <br> markers after headings
  html = html.replace(/###HEADING###<br class="leading-none">/g, "###HEADING###");

  // Remove heading markers
  html = html.replace(/###HEADING###/g, "");

  return html;
}
