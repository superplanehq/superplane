import React from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { showErrorToast } from "@/lib/toast";
import { Checkbox } from "@/ui/checkbox";

import {
  defaultFormValues,
  extractManualRunParams,
  mergeManualRunPayload,
  type ManualRunParamField,
} from "./manualRunParams";

export function StartRunParamsForm({
  templatePayload,
  onRun,
  onClose,
}: {
  templatePayload: Record<string, unknown>;
  onRun: (payload: Record<string, unknown>) => Promise<void>;
  onClose: () => void;
}) {
  const fields = React.useMemo(() => extractManualRunParams(templatePayload), [templatePayload]);
  const [values, setValues] = React.useState<Record<string, unknown>>(() => defaultFormValues(fields));
  const [isSubmitting, setIsSubmitting] = React.useState(false);

  const handleSubmit = async () => {
    const merged = mergeManualRunPayload(templatePayload, values);
    if (merged.error || !merged.payload) {
      showErrorToast(merged.error ?? "Failed to build payload");
      return;
    }

    setIsSubmitting(true);
    try {
      await onRun(merged.payload);
      onClose();
    } catch {
      // Keep the modal open so users can retry.
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="space-y-4">
      <div className="space-y-3">
        {fields.map((field) => (
          <ParamFieldInput
            key={field.path}
            field={field}
            value={values[field.path]}
            onChange={(next) => setValues((prev) => ({ ...prev, [field.path]: next }))}
          />
        ))}
      </div>
      <div className="flex items-center justify-end gap-2">
        <Button variant="outline" onClick={onClose} disabled={isSubmitting}>
          Cancel
        </Button>
        <LoadingButton
          data-testid="emit-event-submit-button"
          loading={isSubmitting}
          loadingText="Running..."
          onClick={handleSubmit}
        >
          Run
        </LoadingButton>
      </div>
    </div>
  );
}

function ParamFieldInput({
  field,
  value,
  onChange,
}: {
  field: ManualRunParamField;
  value: unknown;
  onChange: (value: unknown) => void;
}) {
  const id = `manual-run-param-${field.path.replace(/[^a-zA-Z0-9_-]/g, "-")}`;

  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>
        {field.def.title}
        {field.def.required ? <span className="text-destructive ml-0.5">*</span> : null}
      </Label>
      {field.def.type === "boolean" ? (
        <div className="flex items-center gap-2">
          <Checkbox id={id} checked={Boolean(value)} onCheckedChange={(checked) => onChange(checked === true)} />
        </div>
      ) : field.def.type === "select" ? (
        <Select value={typeof value === "string" ? value : ""} onValueChange={onChange}>
          <SelectTrigger id={id} className="w-full">
            <SelectValue placeholder="Select an option" />
          </SelectTrigger>
          <SelectContent>
            {(field.def.values ?? []).map((option) => (
              <SelectItem key={option} value={option}>
                {option}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      ) : field.def.type === "number" ? (
        <Input
          id={id}
          type="number"
          value={value === undefined || value === null ? "" : String(value)}
          onChange={(e) => onChange(e.target.value === "" ? "" : Number(e.target.value))}
        />
      ) : (
        <Input
          id={id}
          type="text"
          value={typeof value === "string" ? value : value === undefined || value === null ? "" : String(value)}
          onChange={(e) => onChange(e.target.value)}
        />
      )}
    </div>
  );
}
