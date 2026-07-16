import Tippy from "@tippyjs/react/headless";
import type { ReactElement } from "react";
import type { ComponentBaseSpecValue } from "./index";
import "tippy.js/dist/tippy.css";

interface SpecsTooltipProps {
  children: React.ReactNode;
  specTitle: string;
  specValues: ComponentBaseSpecValue[];
  tooltipTitle?: string;
  hideCount?: boolean;
}

const MAX_BADGE_CHUNK_LENGTH = 120;
const BADGE_SEPARATOR_REGEX = /[\s,()[\]{}<>:+\-*/=|&.!?]/;

function splitBadgeLabel(label: string): string[] {
  if (label.length <= MAX_BADGE_CHUNK_LENGTH) {
    return [label];
  }

  const chunks: string[] = [];
  let remaining = label;

  while (remaining.length > MAX_BADGE_CHUNK_LENGTH) {
    let splitAt = -1;
    for (let i = MAX_BADGE_CHUNK_LENGTH - 1; i >= 0; i -= 1) {
      if (BADGE_SEPARATOR_REGEX.test(remaining[i])) {
        splitAt = i + 1;
        break;
      }
    }

    if (splitAt <= 0) {
      splitAt = MAX_BADGE_CHUNK_LENGTH;
    }

    const chunk = remaining.slice(0, splitAt).trim();
    if (chunk.length > 0) {
      chunks.push(chunk);
    }

    remaining = remaining.slice(splitAt).trim();
  }

  if (remaining.length > 0) {
    chunks.push(remaining);
  }

  return chunks;
}

function renderBadgeLabel(badge: ComponentBaseSpecValue["badges"][number], badgeIndex: number) {
  if (!badge) {
    return [];
  }

  const badgeLabel = typeof badge.label === "string" ? badge.label.trim() : "";
  if (badgeLabel.length === 0) {
    return [];
  }

  const chunks = splitBadgeLabel(badgeLabel);
  const shouldStartOnNewLine = badgeLabel.length > MAX_BADGE_CHUNK_LENGTH;

  return chunks.map((chunk, chunkIndex) => (
    <span
      key={`${badgeIndex}-${chunkIndex}`}
      className={`rounded px-2 py-1 font-mono text-xs break-words ${chunkIndex === 0 && shouldStartOnNewLine ? "basis-full" : ""} ${badge.bgColor} ${badge.textColor}`}
      style={{ wordBreak: "break-word", overflowWrap: "break-word" }}
    >
      {chunk}
    </span>
  ));
}

export function SpecsTooltip({ children, specTitle, specValues, tooltipTitle, hideCount }: SpecsTooltipProps) {
  return (
    <Tippy
      render={(attrs) => (
        <div
          {...attrs}
          className="max-w-[800px] overflow-auto rounded-md bg-white shadow-md outline-1 outline-slate-300 dark:bg-gray-900 dark:outline-gray-700"
          style={{ zIndex: 10000, maxHeight: "400px" }}
        >
          <div className="flex items-center border-b border-slate-300 dark:border-gray-700">
            <span className="px-3 py-1.5 text-[13px] font-medium text-gray-500 dark:text-gray-400">
              {!hideCount ? specValues.length : ""} {tooltipTitle || specTitle}
            </span>
          </div>
          {specValues.map((value, index) => (
            <div
              key={index}
              className={`flex max-w-[800px] flex-wrap items-start gap-2 p-2 ${index === specValues.length - 1 ? "border-b-0" : "border-b border-slate-200 dark:border-gray-800"}`}
            >
              {value.badges.flatMap(renderBadgeLabel)}
            </div>
          ))}
        </div>
      )}
      placement="top"
      interactive={true}
      delay={200}
      appendTo={() => document.body}
    >
      {children as ReactElement}
    </Tippy>
  );
}
