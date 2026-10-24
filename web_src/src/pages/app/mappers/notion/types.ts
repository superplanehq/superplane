export interface SearchConfiguration {
  query?: string;
  objectType?: "all" | "page" | "data_source";
  pageSize?: number;
  startCursor?: string;
  sortDirection?: "ascending" | "descending";
}

export interface SearchResult {
  results?: unknown[];
  has_more?: boolean;
  next_cursor?: string | null;
}

export interface CreatePageConfiguration {
  parentType?: "page" | "data_source";
  parentId?: string;
  title?: string;
  properties?: string | Record<string, unknown>;
  children?: string | unknown[];
}

interface RichText {
  plain_text?: string;
  text?: { content?: string };
}

export interface NotionPage {
  id?: string;
  url?: string;
  properties?: Record<string, { type?: string; title?: RichText[] }>;
}
