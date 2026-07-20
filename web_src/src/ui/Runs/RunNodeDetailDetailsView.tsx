import { ArrowRight } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";
import { Timestamp } from "@/components/Timestamp";
import { parseAppRunPath } from "@/lib/appPaths";
import { isUrl } from "@/lib/utils";
import { EventStatusBadge } from "@/ui/EventStatusBadge";
import { isErrorValue } from "./runNodeDetailModel";

const DETAIL_VALUE_PREVIEW_CHARACTER_LIMIT = 160;
const appRunLinkClassName =
  "inline-flex w-fit items-center gap-1 font-medium text-gray-600 underline decoration-gray-300 underline-offset-2 transition-colors hover:text-gray-900 hover:decoration-gray-500 dark:text-gray-400 dark:decoration-gray-600 dark:hover:text-gray-100 dark:hover:decoration-gray-400";

export function RunNodeDetailDetailsView({
  details,
  statusBadge,
  relativeTime,
}: {
  details: Record<string, unknown>;
  statusBadge?: { badgeColor: string; label: string } | null;
  relativeTime?: string;
}) {
  return (
    <div className="flex flex-col gap-1.5 text-[13px]">
      {statusBadge ? (
        <div className="flex items-start gap-2">
          <span className="w-[120px] shrink-0 truncate text-left text-gray-500 dark:text-gray-400">Status:</span>
          <EventStatusBadge badgeColor={statusBadge.badgeColor} label={statusBadge.label} />
        </div>
      ) : null}
      {relativeTime ? (
        <div className="flex items-start gap-2">
          <span className="w-[120px] shrink-0 truncate text-left text-gray-500 dark:text-gray-400">Relative time:</span>
          <span className="min-w-0 break-all text-gray-800 dark:text-gray-100">
            <Timestamp date={relativeTime} display="relative" relativeStyle="abbreviated" />
          </span>
        </div>
      ) : null}
      {Object.entries(details).map(([key, value]) => {
        if (isErrorValue(value)) {
          return (
            <div key={key} className="flex items-start gap-2">
              <span className="w-[120px] shrink-0 truncate text-left text-gray-500 dark:text-gray-400" title={key}>
                {key}:
              </span>
              <span className="min-w-0 break-all font-medium text-red-600 dark:text-red-400">{value.message}</span>
            </div>
          );
        }

        return (
          <div key={key} className="flex items-start gap-2">
            <span className="w-[120px] shrink-0 truncate text-left text-gray-500 dark:text-gray-400" title={key}>
              {key}:
            </span>
            <DetailValue value={value} />
          </div>
        );
      })}
    </div>
  );
}

function DetailValue({ value }: { value: unknown }) {
  const [expanded, setExpanded] = useState(false);
  const stringValue = typeof value === "object" ? JSON.stringify(value, null, 2) : String(value ?? "");
  const canExpand = stringValue.length > DETAIL_VALUE_PREVIEW_CHARACTER_LIMIT;
  const displayValue =
    canExpand && !expanded ? `${stringValue.slice(0, DETAIL_VALUE_PREVIEW_CHARACTER_LIMIT).trimEnd()}...` : stringValue;
  const valueClassName = "min-w-0 break-words whitespace-pre-wrap text-gray-800 dark:text-gray-100";
  const linkClassName =
    "min-w-0 break-words whitespace-pre-wrap text-blue-600 underline underline-offset-2 hover:text-blue-700 dark:text-indigo-300 dark:hover:text-indigo-200";

  const appRunHref = parseAppRunPath(stringValue);
  const content = appRunHref ? (
    <Link to={appRunHref} className={appRunLinkClassName}>
      See run
      <ArrowRight className="h-3.5 w-3.5 shrink-0" aria-hidden />
    </Link>
  ) : isUrl(stringValue) ? (
    <a href={stringValue} target="_blank" rel="noopener noreferrer" className={linkClassName}>
      {displayValue}
    </a>
  ) : (
    <span className={valueClassName}>{displayValue}</span>
  );

  if (!canExpand) {
    return content;
  }

  return (
    <span className="min-w-0 flex-1">
      {content}
      <button
        type="button"
        className="mt-1 block text-xs font-medium text-blue-600 hover:text-blue-700 hover:underline dark:text-indigo-300 dark:hover:text-indigo-200"
        onClick={() => setExpanded((current) => !current)}
      >
        {expanded ? "Collapse" : "Expand"}
      </button>
    </span>
  );
}
