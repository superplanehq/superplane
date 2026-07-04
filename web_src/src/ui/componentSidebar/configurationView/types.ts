export type ConfigurationDisplayKind =
  | "text"
  | "url"
  | "boolean"
  | "expression"
  | "code"
  | "list"
  | "empty"
  | "integration";

export type ConfigurationDisplayRow = {
  key: string;
  label: string;
  kind: ConfigurationDisplayKind;
  /** Plain-text or formatted display value. */
  displayText: string;
  /** When kind is url, the href to link to. */
  href?: string;
  /** Compact chip labels for list-style values. */
  chips?: string[];
  /** Nesting depth for object/list child rows. */
  depth?: number;
  /** Integration status badge label (Ready, Error, etc.). */
  integrationStatus?: string;
  integrationStatusVariant?: "ready" | "error" | "pending";
};

export type ConfigurationDisplayModel = {
  rows: ConfigurationDisplayRow[];
};
