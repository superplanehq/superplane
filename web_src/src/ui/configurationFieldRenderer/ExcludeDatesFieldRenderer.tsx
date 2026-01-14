import React, { useState } from "react";
import { Switch } from "@/ui/switch";
import { Label } from "@/components/ui/label";
import { ListFieldRenderer } from "./ListFieldRenderer";
import { FieldRendererProps, ValidationError } from "./types";
import { AuthorizationDomainType } from "@/api-client";

interface ExcludeDatesFieldRendererProps extends FieldRendererProps {
  domainId?: string;
  domainType?: AuthorizationDomainType;
  validationErrors?: ValidationError[] | Set<string>;
  fieldPath?: string;
}

export const ExcludeDatesFieldRenderer: React.FC<ExcludeDatesFieldRendererProps> = ({
  field,
  value,
  onChange,
  domainId,
  domainType,
  validationErrors,
  fieldPath,
  hasError = false,
}) => {
  const [isEnabled, setIsEnabled] = useState(
    Array.isArray(value) && value.length > 0
  );

  const handleToggleChange = (checked: boolean) => {
    setIsEnabled(checked);
    if (!checked) {
      // When toggling off, clear the exclude dates
      onChange(undefined);
    } else if (!value || (Array.isArray(value) && value.length === 0)) {
      // When toggling on, initialize with empty array
      onChange([]);
    }
  };

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-3">
        <Switch
          checked={isEnabled}
          onCheckedChange={handleToggleChange}
          className={hasError ? "border-red-500 border-2" : ""}
        />
        <Label className={`block text-left ${hasError ? "text-red-600 dark:text-red-400" : ""}`}>
          {field.label || field.name}
          {field.required && <span className="text-red-500 ml-1">*</span>}
        </Label>
      </div>
      {isEnabled && (
        <div className="flex items-center gap-2">
          <div className="flex-1">
            <ListFieldRenderer
              field={field}
              value={value}
              onChange={onChange}
              domainId={domainId}
              domainType={domainType}
              validationErrors={validationErrors}
              fieldPath={fieldPath}
              hasError={hasError}
            />
          </div>
        </div>
      )}
    </div>
  );
};
