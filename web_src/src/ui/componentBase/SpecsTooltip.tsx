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
          className="bg-white dark:bg-slate-800 outline-1 outline-slate-300 dark:outline-slate-600 shadow-md rounded-md max-w-[1400px]"
          style={{ zIndex: 10000 }}
        >
          <div className="flex items-center border-b border-slate-300 dark:border-slate-600">
            <span className="font-medium text-gray-500 dark:text-gray-300 text-[13px] px-3 py-1.5">
              {!hideCount ? specValues.length : ""} {tooltipTitle || specTitle}
            </span>
          </div>
          {specValues.map((value, index) => (
            <div
              key={index}
              className={`flex max-w-[1400px] items-center gap-2 p-2 overflow-x-auto ${index === specValues.length - 1 ? "border-b-0" : "border-b border-slate-300 dark:border-slate-600"}`}
            >
              {value.badges.map((badge, badgeIndex) => (
                <span
                  key={badgeIndex}
                  className={`px-2 py-1 rounded text-xs font-mono whitespace-nowrap ${badge.bgColor} ${badge.textColor}`}
                >
                  {badge.label}
                </span>
              ))}
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
