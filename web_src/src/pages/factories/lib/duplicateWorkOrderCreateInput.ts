const FILE_REF_PREFIX = "sp-file://";
const MARKDOWN_LINK = /(!?\[[^\]]*]\()([^)\s]+)(\))/g;
const HTML_IMAGE = /<img\b[^>]*?\bsrc\s*=\s*(["'])([^"']+)\1[^>]*\/?>/gi;
const HTML_ANCHOR = /<a\b[^>]*?\bhref\s*=\s*(["'])([^"']+)\1[^>]*>[\s\S]*?<\/a>/gi;

export interface DuplicateWorkOrderSource {
  title?: string | null;
  description?: string | null;
}

export interface DuplicateWorkOrderCreateInput {
  title: string;
  description: string;
}

export function duplicateWorkOrderCreateInput(source: DuplicateWorkOrderSource): DuplicateWorkOrderCreateInput {
  return {
    title: source.title ?? "",
    description: descriptionWithoutFileLinks(source.description ?? ""),
  };
}

function descriptionWithoutFileLinks(description: string): string {
  return tidyDescription(
    description
      .replace(MARKDOWN_LINK, (full, _open: string, target: string) => (isFileTarget(target) ? "" : full))
      .replace(HTML_IMAGE, (full, _quote: string, target: string) => (isFileTarget(target) ? "" : full))
      .replace(HTML_ANCHOR, (full, _quote: string, target: string) => (isFileTarget(target) ? "" : full)),
  );
}

function isFileTarget(target: string): boolean {
  return target.trim().startsWith(FILE_REF_PREFIX);
}

function tidyDescription(description: string): string {
  return description
    .replace(/[^\S\n]{2,}/g, " ")
    .replace(/\n{3,}/g, "\n\n")
    .trim();
}
