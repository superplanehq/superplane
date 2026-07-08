import React, { useEffect, useMemo } from "react";
import type { FieldRendererProps } from "./types";
import { MultiCombobox, MultiComboboxLabel } from "@/components/MultiCombobox/multi-combobox";
import { Checkbox } from "@/ui/checkbox";
import { Label } from "@/components/ui/label";

interface SelectOption {
  id: string;
  label: string;
  value: string;
  description?: string;
}

function parseMultiSelectValues(raw: unknown): string[] {
  if (Array.isArray(raw)) {
    return raw.filter((item): item is string => typeof item === "string");
  }

  if (typeof raw !== "string" || raw === "") {
    return [];
  }

  try {
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      return [];
    }

    return parsed.filter((item): item is string => typeof item === "string");
  } catch {
    return [];
  }
}

function toCheckboxOptionId(fieldName: string, optionValue: string): string {
  const safeFieldName = fieldName.replace(/[^a-zA-Z0-9_-]/g, "-");
  const safeOptionValue = optionValue.replace(/[^a-zA-Z0-9_-]/g, "-");
  return `${safeFieldName}-${safeOptionValue}`;
}

export const MultiSelectFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  readOnly = false,
}) => {
  const useCheckboxes = field.typeOptions?.multiSelect?.useCheckboxes === true;

  const comboboxOptions: SelectOption[] = useMemo(() => {
    const multiSelectOptions = field.typeOptions?.multiSelect?.options ?? [];
    const options: SelectOption[] = [];
    for (const opt of multiSelectOptions) {
      if (!opt.value) {
        continue;
      }

      options.push({
        id: opt.value,
        label: opt.label ?? opt.value,
        value: opt.value,
        description: opt.description,
      });
    }

    return options;
  }, [field.typeOptions?.multiSelect?.options]);

  const defaultValues = useMemo(() => parseMultiSelectValues(field.defaultValue), [field.defaultValue]);

  // Set initial value on first render if no value is present but there's a default
  useEffect(() => {
    if (readOnly) return;

    if ((value === undefined || value === null) && defaultValues.length > 0) {
      onChange(defaultValues);
    }
  }, [readOnly, value, defaultValues, onChange]);

  const currentValue = useMemo(() => {
    if (value === undefined || value === null) {
      return defaultValues;
    }

    return parseMultiSelectValues(value);
  }, [value, defaultValues]);
  const selectedValues = useMemo(() => new Set(currentValue), [currentValue]);

  // Convert selected values to SelectOption objects
  const selectedOptions: SelectOption[] = currentValue.map((val) => {
    const option = comboboxOptions.find((opt) => opt.value === val);
    return option || { id: val, label: val, value: val };
  });

  const handleComboboxChange = (nextSelectedOptions: SelectOption[]) => {
    const nextSelectedValues = nextSelectedOptions.map((opt) => opt.value);
    onChange(nextSelectedValues.length > 0 ? nextSelectedValues : undefined);
  };

  const handleCheckboxChange = (selectedOptionValue: string, checked: boolean) => {
    const nextSelectedValues = checked
      ? Array.from(new Set([...currentValue, selectedOptionValue]))
      : currentValue.filter((selectedValue) => selectedValue !== selectedOptionValue);

    onChange(nextSelectedValues.length > 0 ? nextSelectedValues : undefined);
  };

  if (useCheckboxes) {
    return (
      <div className="space-y-2">
        {comboboxOptions.map((option) => {
          const optionId = toCheckboxOptionId(field.name ?? "multi-select", option.value);

          return (
            <div key={option.id} className="rounded-md border border-border/70 px-3 py-2">
              <div className="flex items-start gap-3">
                <Checkbox
                  id={optionId}
                  checked={selectedValues.has(option.value)}
                  onCheckedChange={(checked) => handleCheckboxChange(option.value, checked === true)}
                  disabled={readOnly}
                  className="mt-0.5"
                />
                <div className="flex flex-col gap-1 py-0.5">
                  <Label
                    htmlFor={optionId}
                    className={readOnly ? "font-medium leading-none" : "cursor-pointer font-medium leading-none"}
                  >
                    {option.label}
                  </Label>
                  {option.description && (
                    <p className="text-xs leading-relaxed text-muted-foreground">{option.description}</p>
                  )}
                </div>
              </div>
            </div>
          );
        })}
      </div>
    );
  }

  return (
    <MultiCombobox<SelectOption>
      options={comboboxOptions}
      displayValue={(option) => option.label}
      placeholder={`Select ${field.label || field.name}...`}
      value={selectedOptions}
      onChange={readOnly ? undefined : handleComboboxChange}
      showButton={false}
      disabled={readOnly}
    >
      {(option) => <MultiComboboxLabel>{option.label}</MultiComboboxLabel>}
    </MultiCombobox>
  );
};
