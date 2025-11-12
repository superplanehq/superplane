import React from "react";
import Tippy from "@tippyjs/react/headless";
import "tippy.js/dist/tippy.css";

interface SimpleTooltipProps {
  children: React.ReactElement;
  content: string;
  delay?: number;
  hideOnClick?: boolean;
}

export const SimpleTooltip: React.FC<SimpleTooltipProps> = ({ children, content, delay = 200, hideOnClick = true }) => {
  return (
    <Tippy
      render={() => <div className="bg-gray-800 text-white text-xs px-2 py-1 rounded shadow-lg">{content}</div>}
      placement="top"
      delay={delay}
      hideOnClick={hideOnClick}
    >
      {children}
    </Tippy>
  );
};
