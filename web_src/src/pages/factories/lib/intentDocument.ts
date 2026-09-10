export const INTENT_ARTIFACT_NAME = "intent.md";

export const INTENT_DOCUMENT_TITLE = "What this work will do";

export const EXECUTIVE_SUMMARY_HEADING = "Executive summary";

export interface IntentDocument {
  title: string;
  summary: string;
  plan: string;
}

export function parseIntentDocument(markdown: string): IntentDocument {
  const trimmed = markdown.trim();
  if (!trimmed) {
    return { title: INTENT_DOCUMENT_TITLE, summary: "", plan: "" };
  }

  const titled = trimmed.match(/^#\s+(.+)\n+([\s\S]*)$/);
  const title = titled?.[1]?.trim() || INTENT_DOCUMENT_TITLE;
  const body = (titled?.[2] ?? trimmed).trim();

  const executive = headingSection(body, EXECUTIVE_SUMMARY_HEADING);
  if (executive) {
    return {
      title,
      summary: executive.content,
      plan: stripSection(body, executive).trim() || body,
    };
  }

  const headings = [...body.matchAll(/^##\s+.+$/gm)];
  if (headings.length === 0) {
    return { title, summary: firstParagraph(body), plan: body };
  }

  const first = headings[0];
  const start = first.index! + first[0].length;
  const end = headings[1]?.index ?? body.length;
  const summary = body.slice(start, end).trim();
  const plan = headings[1] ? body.slice(headings[1].index!).trim() : body;
  return { title, summary, plan };
}

export function composeIntentDocument(description: string, plan: string): IntentDocument {
  const summary = firstParagraph(description);
  const planBody = plan.trim() || description.trim();
  return {
    title: INTENT_DOCUMENT_TITLE,
    summary,
    plan: planBody,
  };
}

export function firstParagraph(markdown: string): string {
  const block = markdown.trim().split(/\n\s*\n/)[0] ?? "";
  return block.replace(/^#+\s+/, "").trim();
}

function headingSection(body: string, heading: string): { content: string; start: number; end: number } | undefined {
  const match = body.match(new RegExp(`^##\\s+${escapeRegExp(heading)}\\s*$`, "im"));
  if (!match || match.index == null) {
    return undefined;
  }
  const contentStart = match.index + match[0].length;
  const rest = body.slice(contentStart);
  const next = rest.match(/^##\s+/m);
  const contentEnd = next?.index != null ? contentStart + next.index : body.length;
  return {
    content: body.slice(contentStart, contentEnd).trim(),
    start: match.index,
    end: contentEnd,
  };
}

function stripSection(body: string, section: { start: number; end: number }): string {
  return `${body.slice(0, section.start)}${body.slice(section.end)}`;
}

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
