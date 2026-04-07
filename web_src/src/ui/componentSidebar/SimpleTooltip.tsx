import React from "react";
import Tippy from "@tippyjs/react/headless";
import "tippy.js/dist/tippy.css";

interface SimpleTooltipProps {
  children: React.ReactElement;
  content: React.ReactNode;
  delay?: number;
  hideOnClick?: boolean;
}

export const SimpleTooltip: React.FC<SimpleTooltipProps> = ({ children, content, delay = 200, hideOnClick = true }) => {
  return (
    <Tippy
      render={() => (
        <div className="bg-gray-800 text-white text-xs px-2 py-1 rounded shadow-lg w-[90vw] max-w-[520px] max-h-[40vh] overflow-auto whitespace-pre-wrap break-words">
          {content}
        </div>
      )}
      placement="top"
      delay={delay}
      hideOnClick={hideOnClick}
    >
      {children}
    </Tippy>
  );
};
