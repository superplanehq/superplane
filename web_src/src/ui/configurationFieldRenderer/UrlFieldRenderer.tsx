import React from "react";
import { Input } from "@/components/ui/input";
import { FieldRendererProps } from "./types";
import { InlineFieldAssistant } from "@/ui/InlineFieldAssistant";

export const UrlFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  suggestFieldValue,
  assistantEnabled = false,
  labelRightRef,
  labelRightReady = false,
}) => {
  const fieldLabel = field.label || field.name || "Field";
  const showAssistant = Boolean(assistantEnabled && suggestFieldValue && !field.sensitive && field.type === "url");

  return (
    <div className="space-y-2">
      {showAssistant ? (
        <InlineFieldAssistant
          fieldLabel={fieldLabel}
          onApplyValue={(next) => onChange(next.trim() || undefined)}
          suggestFieldValue={suggestFieldValue}
          assistantEnabled={assistantEnabled}
          labelRightRef={labelRightRef}
          labelRightReady={labelRightReady}
        />
      ) : null}
      <Input
        type="url"
        value={(value as string) ?? (field.defaultValue as string) ?? ""}
        onChange={(e) => onChange(e.target.value || undefined)}
        placeholder={field.placeholder || `https://example.com`}
        className=""
      />
    </div>
  );
};
