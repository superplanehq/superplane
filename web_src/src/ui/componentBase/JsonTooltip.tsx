import Tippy from "@tippyjs/react/headless";
import { ReactElement } from "react";
import JsonView from "@uiw/react-json-view";
import { lightTheme } from "@uiw/react-json-view/light";
import "tippy.js/dist/tippy.css";

interface JsonTooltipProps {
  children: React.ReactNode;
  title: string;
  value: Record<string, any>;
}

export function JsonTooltip({ children, title, value }: JsonTooltipProps) {
  return (
    <Tippy
      render={() => (
        <div className="bg-white border-2 border-gray-200 rounded-md max-w-[500px] max-h-[400px] overflow-auto text-left">
          <div className="flex items-center border-b-2 p-2">
            <span className="font-medium text-gray-500 text-sm">{title}</span>
          </div>
          <div className="p-2">
            <JsonView
              value={value}
              style={{
                fontSize: "12px",
                fontFamily:
                  'ui-monospace, SFMono-Regular, "SF Mono", Consolas, "Liberation Mono", Menlo, monospace',
                backgroundColor: "transparent",
                textAlign: "left",
                ...lightTheme,
              }}
              displayDataTypes={false}
              displayObjectSize={false}
              enableClipboard={false}
              collapsed={1}
            />
          </div>
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
