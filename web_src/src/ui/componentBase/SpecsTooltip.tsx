import Tippy from '@tippyjs/react/headless';
import { ReactElement } from 'react';
import { ComponentBaseSpecValue } from './index';
import 'tippy.js/dist/tippy.css';


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
      render={() => (
        <div className="bg-white border-2 border-gray-200 rounded-md max-w-[700px]">
          <div className="flex items-center  border-b-2 p-2">
            <span className="font-medium text-gray-500 text-sm">{!hideCount ? specValues.length : ''} {tooltipTitle || specTitle}</span>
          </div>
          {
            specValues.map((value, index) => (
              <div key={index} className={`flex max-w-[700px] items-center gap-2 p-2 ${index === specValues.length - 1 ? "border-b-0" : "border-b-2"}`}>
                {value.badges.map((badge, badgeIndex) => (
                  <span key={badgeIndex} className={`px-2 py-1 rounded-md text-sm font-mono font-medium whitespace-nowrap ${badge.bgColor} ${badge.textColor}`}>{badge.label}</span>
                ))}
              </div>
            ))
          }
        </div>
      )}
      placement="top"
      interactive={true}
      delay={200}
    >
      {children as ReactElement}
    </Tippy>
  );
}