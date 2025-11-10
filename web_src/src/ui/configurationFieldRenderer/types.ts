import { ConfigurationField } from "../../api-client";

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
  domainType?: "DOMAIN_TYPE_CANVAS" | "DOMAIN_TYPE_ORGANIZATION";
  hasError?: boolean;
  validationErrors?: ValidationError[] | Set<string>;
  fieldPath?: string;
}
