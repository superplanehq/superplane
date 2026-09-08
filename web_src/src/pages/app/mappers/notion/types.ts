/** Shapes read on notion.onPageAdded nodes. */

/** A Notion rich text run, trimmed to the field every mapper reads. */
export interface NotionRichText {
  plain_text?: string;
}

/** A Notion page property, trimmed to the title property shape. */
export interface NotionProperty {
  type?: string;
  title?: NotionRichText[];
}

/** A Notion page, as the onPageAdded trigger emits it: every field Notion
 * returned, plus a simplified title and content. */
export interface NotionPage {
  id?: string;
  url?: string;
  created_time?: string;
  properties?: Record<string, NotionProperty>;
  title?: string;
  content?: string;
}

/** Envelope the onPageAdded trigger emits. */
export interface NotionPageEnvelope {
  meta?: { event?: string };
  data?: NotionPage;
}

export interface NotionDatabase {
  id?: string;
  name?: string;
}

/** Metadata SuperPlane stores on Notion nodes during setup. */
export interface NotionNodeMetadata {
  database?: NotionDatabase;
}

export interface OnPageAddedConfiguration {
  database?: string;
}
