import React, { useEffect, useMemo, useRef } from "react";
import { AutoCompleteSelect } from "@/components/AutoCompleteSelect/AutoCompleteSelect";
import type { FieldRendererProps } from "./types";
import { toTestId } from "@/lib/testID";

const FALLBACK_TIMEZONES = [
  "UTC",
  "America/Anchorage",
  "America/Chicago",
  "America/Denver",
  "America/Los_Angeles",
  "America/New_York",
  "America/Phoenix",
  "America/Sao_Paulo",
  "America/Toronto",
  "Asia/Dubai",
  "Asia/Kolkata",
  "Asia/Shanghai",
  "Asia/Singapore",
  "Asia/Tokyo",
  "Australia/Melbourne",
  "Australia/Sydney",
  "Europe/Berlin",
  "Europe/Dublin",
  "Europe/London",
  "Europe/Madrid",
  "Europe/Paris",
  "Pacific/Auckland",
  "Pacific/Honolulu",
];

/**
 * Every IANA timezone the browser knows about, falling back to a short list on
 * engines without Intl.supportedValuesOf.
 */
function listTimezones(): string[] {
  const supportedValuesOf = (Intl as { supportedValuesOf?: (key: string) => string[] }).supportedValuesOf;

  if (typeof supportedValuesOf === "function") {
    try {
      const zones = supportedValuesOf("timeZone");
      if (zones.length > 0) {
        return zones;
      }
    } catch {
      // fall through to the static list
    }
  }

  return FALLBACK_TIMEZONES;
}

function getBrowserTimezone(): string {
  return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
}

/**
 * Current UTC offset for a timezone, e.g. "GMT-4". Shown next to the identifier
 * so the list stays scannable. It reflects daylight saving, so the same zone can
 * read differently depending on the time of year.
 */
function formatOffset(timeZone: string): string {
  try {
    const parts = new Intl.DateTimeFormat("en-US", {
      timeZone,
      timeZoneName: "shortOffset",
    }).formatToParts(new Date());

    return parts.find((part) => part.type === "timeZoneName")?.value ?? "";
  } catch {
    return "";
  }
}

/**
 * Legacy configurations stored a bare UTC offset such as "-5" instead of an
 * identifier. Those values still work, so they are kept selectable rather than
 * silently replaced.
 */
function isLegacyOffset(value: string): boolean {
  return /^[+-]?\d+(\.\d+)?$/.test(value);
}

export const TimezoneFieldRenderer: React.FC<FieldRendererProps> = ({ field, value, onChange, hasError }) => {
  const hasSetDefault = useRef(false);
  const testId = field.name ? toTestId(`field-${field.name}-select`) : undefined;

  const options = useMemo(() => {
    const timezones = listTimezones().map((timeZone) => {
      const offset = formatOffset(timeZone);
      const [group] = timeZone.split("/");

      return {
        value: timeZone,
        label: offset ? `${timeZone} (${offset})` : timeZone,
        group: timeZone.includes("/") ? group : "Other",
      };
    });

    //
    // Keep a stored legacy offset visible so opening an existing configuration
    // does not blank the field.
    //
    const current = typeof value === "string" ? value : "";
    if (current && isLegacyOffset(current)) {
      timezones.unshift({
        value: current,
        label: `GMT${current.startsWith("-") ? current : `+${current.replace("+", "")}`} (fixed offset)`,
        group: "Other",
      });
    }

    return timezones;
  }, [value]);

  //
  // "current" is a placeholder the backend rejects, so it is resolved to the
  // browser's timezone before anything is submitted.
  //
  useEffect(() => {
    if (hasSetDefault.current) return;

    if (value === undefined || value === null || value === "current") {
      onChange(getBrowserTimezone());
      hasSetDefault.current = true;
    }
  }, [value, onChange]);

  const displayValue = typeof value === "string" && value !== "current" ? value : "";

  return (
    <div data-testid={testId}>
      <AutoCompleteSelect
        options={options}
        value={displayValue}
        onChange={onChange}
        placeholder={`Select ${field.label || field.name}`}
        error={hasError}
      />
    </div>
  );
};
