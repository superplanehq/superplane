import { TimeAgo } from "@/components/TimeAgo";
import { cn, isUrl } from "@/lib/utils";
import { isErrorValue } from "./runNodeDetailModel";

/** Matches {@link EventSectionDisplay} status chip on canvas nodes (style + casing). */
function EventSectionStatusBadge({ badgeColor, label }: { badgeColor: string; label: string }) {
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center justify-center rounded px-[5px] py-[1.5px] text-[10px] font-semibold uppercase tracking-wide text-white",
        badgeColor,
      )}
    >
      {label}
    </span>
  );
}

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
          <span className="w-[120px] shrink-0 truncate text-right text-gray-500">Status:</span>
          <EventSectionStatusBadge badgeColor={statusBadge.badgeColor} label={statusBadge.label} />
        </div>
      ) : null}
      {relativeTime ? (
        <div className="flex items-start gap-2">
          <span className="w-[120px] shrink-0 truncate text-right text-gray-500">Relative time:</span>
          <span className="min-w-0 break-all text-gray-800">
            <TimeAgo date={relativeTime} />
          </span>
        </div>
      ) : null}
      {Object.entries(details).map(([key, value]) => {
        if (isErrorValue(value)) {
          return (
            <div key={key} className="flex items-start gap-2">
              <span className="w-[120px] shrink-0 truncate text-right text-gray-500" title={key}>
                {key}:
              </span>
              <span className="min-w-0 break-all font-medium text-red-600">{value.message}</span>
            </div>
          );
        }

        return (
          <div key={key} className="flex items-start gap-2">
            <span className="w-[120px] shrink-0 truncate text-right text-gray-500" title={key}>
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
  const stringValue = typeof value === "object" ? JSON.stringify(value, null, 2) : String(value ?? "");

  if (isUrl(stringValue)) {
    return (
      <a
        href={stringValue}
        target="_blank"
        rel="noopener noreferrer"
        className="min-w-0 break-all text-blue-600 underline underline-offset-2 hover:text-blue-700"
      >
        {stringValue}
      </a>
    );
  }

  return <span className="min-w-0 break-all text-gray-800">{stringValue}</span>;
}
