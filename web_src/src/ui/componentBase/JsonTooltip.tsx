import Tippy from "@tippyjs/react/headless";
import { ReactElement } from "react";
import JsonView from "@uiw/react-json-view";
import { lightTheme } from "@uiw/react-json-view/light";
import { darkTheme } from "@uiw/react-json-view/dark";
import { useTheme } from "@/hooks/useTheme";
import "tippy.js/dist/tippy.css";

interface JsonTooltipProps {
  children: React.ReactNode;
  title: string;
  value: Record<string, any>;
}

export function JsonTooltip({ children, title, value }: JsonTooltipProps) {
  const { isDark } = useTheme();

  return (
    <Tippy
      render={() => (
        <div className="bg-white dark:bg-slate-800 border-2 border-gray-200 dark:border-slate-600 rounded-md max-w-[500px] max-h-[400px] overflow-auto text-left">
          <div className="flex items-center border-b border-gray-200 dark:border-slate-600 p-2">
            <span className="font-medium text-gray-500 dark:text-gray-300 text-sm">{title}</span>
          </div>
          <div className="p-2">
            <JsonView
              value={value}
              style={{
                fontSize: "12px",
                fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Consolas, "Liberation Mono", Menlo, monospace',
                backgroundColor: "transparent",
                textAlign: "left",
                ...(isDark ? darkTheme : lightTheme),
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
