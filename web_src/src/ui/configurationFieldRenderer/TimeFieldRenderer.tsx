import React, { useEffect, useRef } from "react";
import { Input } from "@/components/ui/input";
import type { FieldRendererProps } from "./types";
import { toTestId } from "@/lib/testID";
import { useSkipDefaultsAfterReadOnly } from "./useSkipDefaultsAfterReadOnly";

export const TimeFieldRenderer: React.FC<FieldRendererProps> = ({
  field,
  value,
  onChange,
  allValues = {},
  readOnly = false,
}) => {
  const hasAppliedDefault = useRef(false);
  const skipDefaultsAfterReadOnly = useSkipDefaultsAfterReadOnly(readOnly);

  useEffect(() => {
    if (readOnly || skipDefaultsAfterReadOnly || hasAppliedDefault.current) {
      return;
    }

    if ((value === undefined || value === null) && field.defaultValue !== undefined) {
      const defaultVal = field.defaultValue as string;
      if (defaultVal && defaultVal !== "") {
        hasAppliedDefault.current = true;
        onChange(defaultVal);
      }
    }
  }, [readOnly, skipDefaultsAfterReadOnly, value, field.defaultValue, onChange]);

  // Calculate min/max based on other fields for time gates
  const getTimeConstraints = React.useMemo(() => {
    // For endTime field in time gates, prevent selecting times before startTime
    if (field.name === "endTime" && allValues.startTime) {
      return { min: allValues.startTime as string };
    }

    // For startTime field in time gates, prevent selecting times after endTime
    if (field.name === "startTime" && allValues.endTime) {
      return { max: allValues.endTime as string };
    }

    return {};
  }, [field.name, allValues.startTime, allValues.endTime]);

  return (
    <Input
      type="time"
      value={(value as string) ?? (field.defaultValue as string) ?? ""}
      onChange={(e) => onChange(e.target.value || undefined)}
      placeholder={field.typeOptions?.time?.format || "HH:MM"}
      className=""
      min={getTimeConstraints.min}
      max={getTimeConstraints.max}
      disabled={readOnly}
      data-testid={toTestId(`time-field-${field.name}`)}
    />
  );
};
