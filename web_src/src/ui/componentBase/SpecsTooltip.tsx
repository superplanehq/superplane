import Tippy from "@tippyjs/react/headless";
import { ReactElement } from "react";
import { ComponentBaseSpecValue } from "./index";
import "tippy.js/dist/tippy.css";

interface SpecsTooltipProps {
  children: React.ReactNode;
  specTitle: string;
  specValues: ComponentBaseSpecValue[];
  tooltipTitle?: string;
  hideCount?: boolean;
}

export function SpecsTooltip({ children, specTitle, specValues, tooltipTitle, hideCount }: SpecsTooltipProps) {
  return (
    <Tippy
      render={(attrs) => (
        <div
          {...attrs}
          className="bg-white outline-1 outline-slate-300 shadow-md rounded-md max-w-[800px] overflow-auto"
          style={{ zIndex: 10000, maxHeight: "400px" }}
        >
          <div className="flex items-center border-b border-slate-300">
            <span className="font-medium text-gray-500 text-[13px] px-3 py-1.5">
              {!hideCount ? specValues.length : ""} {tooltipTitle || specTitle}
            </span>
          </div>
          {specValues.map((value, index) => (
            <div
              key={index}
              className={`flex flex-wrap max-w-[800px] items-start gap-2 p-2 ${index === specValues.length - 1 ? "border-b-0" : "border-b"}`}
            >
              {value.badges.flatMap((badge, badgeIndex) => {
                const maxChunkLength = 120;
                if (badge.label.length <= maxChunkLength) {
                  return (
                    <span
                      key={`${badgeIndex}-0`}
                      className={`px-2 py-1 rounded text-xs font-mono break-words ${badge.bgColor} ${badge.textColor}`}
                      style={{ wordBreak: "break-word", overflowWrap: "break-word" }}
                    >
                      {badge.label}
                    </span>
                  );
                }

                const separatorRegex = /[\s,()[\]{}<>:+\-*/=|&.!?]/;
                const chunks: string[] = [];
                let remaining = badge.label;

                while (remaining.length > maxChunkLength) {
                  let splitAt = -1;
                  for (let i = maxChunkLength - 1; i >= 0; i -= 1) {
                    if (separatorRegex.test(remaining[i])) {
                      splitAt = i + 1;
                      break;
                    }
                  }

                  if (splitAt <= 0) {
                    splitAt = maxChunkLength;
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

                return chunks.map((chunk, chunkIndex) => (
                  <span
                    key={`${badgeIndex}-${chunkIndex}`}
                    className={`px-2 py-1 rounded text-xs font-mono break-words ${chunkIndex === 0 ? "basis-full" : ""} ${badge.bgColor} ${badge.textColor}`}
                    style={{ wordBreak: "break-word", overflowWrap: "break-word" }}
                  >
                    {chunk}
                  </span>
                ));
              })}
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
