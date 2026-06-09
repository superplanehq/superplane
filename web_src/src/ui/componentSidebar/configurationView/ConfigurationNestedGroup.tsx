import type { ReactNode } from "react";
import { cn } from "@/lib/utils";
import type { ConfigurationDisplayBlock } from "./configurationDisplayBlocks";
import type { ConfigurationDisplayRow } from "./types";
import { EMPTY_DISPLAY_VALUE } from "./formatConfigurationValue";

type ConfigurationGroupHeaderProps = {
  header: ConfigurationDisplayRow;
  className?: string;
};

function ConfigurationGroupHeader({ header, className }: ConfigurationGroupHeaderProps) {
  const hasSummary = header.displayText !== "" && header.displayText !== EMPTY_DISPLAY_VALUE;

  return (
    <div className={cn("flex items-baseline justify-between gap-2", className)}>
      <p className="text-[12px] font-semibold text-gray-700 dark:text-gray-200">{header.label}</p>
      {hasSummary ? <span className="text-[11px] text-gray-500 dark:text-gray-400">{header.displayText}</span> : null}
    </div>
  );
}

type ConfigurationNestedGroupProps = {
  header: ConfigurationDisplayRow;
  children: ReactNode;
  className?: string;
  contentClassName?: string;
};

export function ConfigurationNestedGroup({
  header,
  children,
  className,
  contentClassName,
}: ConfigurationNestedGroupProps) {
  return (
    <div className={cn("min-w-0 mt-1", className)}>
      <ConfigurationGroupHeader header={header} />
      <div
        className={cn(
          "relative ml-1.5 mt-1.5 min-w-0 border-l border-slate-950/10 pl-3",
          contentClassName,
        )}
      >
        <div className="flex flex-col gap-2">{children}</div>
      </div>
    </div>
  );
}

type ConfigurationDisplayBlockListProps = {
  blocks: ConfigurationDisplayBlock[];
  renderRow: (row: ConfigurationDisplayRow) => ReactNode;
};

export function ConfigurationDisplayBlockList({ blocks, renderRow }: ConfigurationDisplayBlockListProps) {
  return (
    <>
      {blocks.map((block) => {
        if (block.type === "row") {
          return <div key={block.row.key}>{renderRow(block.row)}</div>;
        }

        return (
          <ConfigurationNestedGroup key={block.header.key} header={block.header}>
            <ConfigurationDisplayBlockList blocks={block.children} renderRow={renderRow} />
          </ConfigurationNestedGroup>
        );
      })}
    </>
  );
}
