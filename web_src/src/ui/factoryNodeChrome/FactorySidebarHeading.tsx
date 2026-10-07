import { cn } from "@/lib/utils";
import { resolveFactorySidebarHeading } from "./resolveFactorySidebarHeading";

export function FactorySidebarHeading({
  componentLabel,
  nodeName,
  className,
  testId,
  as: Tag = "h2",
}: {
  componentLabel?: string;
  nodeName?: string;
  className?: string;
  testId?: string;
  as?: "h2" | "h3";
}) {
  const { primary, secondary } = resolveFactorySidebarHeading({ componentLabel, nodeName });

  return (
    <Tag
      className={cn(
        "min-w-0 flex-1 truncate text-[13px] font-semibold tracking-[-0.01em] text-slate-900 dark:text-gray-100",
        className,
      )}
      data-testid={testId}
    >
      <span>{primary}</span>
      {secondary ? (
        <>
          <span className="font-normal text-muted-foreground"> · </span>
          <span className="font-normal text-muted-foreground">{secondary}</span>
        </>
      ) : null}
    </Tag>
  );
}
