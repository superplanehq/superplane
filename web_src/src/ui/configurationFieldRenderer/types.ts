import type { RefObject } from "react";
import type { AuthorizationDomainType, ConfigurationField } from "../../api-client";

export type FieldSuggestResult = { value: string; explanation?: string };

export type SuggestFieldValueFn = (instruction: string) => Promise<FieldSuggestResult>;

export interface ValidationError {
  field: string;
  message: string;
  type: "required" | "validation_rule" | "format";
}

export interface FieldRendererProps {
  field: ConfigurationField;
  value: unknown;
  onChange: (value: unknown) => void;
  allValues?: Record<string, unknown>;
  domainId?: string;
  domainType?: AuthorizationDomainType;
  integrationId?: string;
  organizationId?: string;
  hasError?: boolean;
  validationErrors?: ValidationError[] | Set<string>;
  fieldPath?: string;
  autocompleteExampleObj?: Record<string, unknown> | null;
  allowExpressions?: boolean;
  suggestFieldValue?: SuggestFieldValueFn;
  labelRightRef?: RefObject<HTMLDivElement | null>;
  labelRightReady?: boolean;
}
