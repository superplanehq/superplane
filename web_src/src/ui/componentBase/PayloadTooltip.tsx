import Tippy from "@tippyjs/react/headless";
import type { ReactElement } from "react";
import Editor from "@monaco-editor/react";
import { useTheme } from "@/contexts/useTheme";
import "tippy.js/dist/tippy.css";

interface PayloadTooltipProps {
  children: React.ReactNode;
  title: string;
  value: unknown;
  contentType?: "json" | "xml" | "text";
}

interface PayloadEditorProps {
  contentType: NonNullable<PayloadTooltipProps["contentType"]>;
  monacoTheme: string;
  value: unknown;
}

function stringifyPayloadValue(value: unknown): string {
  if (typeof value === "string") {
    return value;
  }

  return JSON.stringify(value, null, 2) ?? "";
}

function payloadLanguage(contentType: PayloadEditorProps["contentType"]): "json" | "xml" | "plaintext" {
  if (contentType === "xml") {
    return "xml";
  }

  if (contentType === "json") {
    return "json";
  }

  return "plaintext";
}

function PayloadEditor({ contentType, monacoTheme, value }: PayloadEditorProps) {
  const stringValue = stringifyPayloadValue(value);
  const lineCount = stringValue.split("\n").length;
  const lineHeight = 19;
  const calculatedHeight = Math.min(Math.max(lineCount * lineHeight + 10, 100), 350);
  const language = payloadLanguage(contentType);

  return (
    <div className="overflow-hidden" style={{ height: `${calculatedHeight}px`, width: "100%", minWidth: "300px" }}>
      <Editor
        height="100%"
        width="100%"
        defaultLanguage={language}
        value={stringValue}
        theme={monacoTheme}
        options={{
          readOnly: true,
          minimap: { enabled: false },
          fontSize: 12,
          lineNumbers: "off",
          wordWrap: "on",
          folding: language !== "plaintext",
          scrollBeyondLastLine: false,
          renderWhitespace: "none",
          contextmenu: true,
          cursorStyle: "line",
          scrollbar: {
            vertical: "auto",
            horizontal: "auto",
          },
        }}
      />
    </div>
  );
}

export function PayloadTooltip({ children, title, value, contentType = "json" }: PayloadTooltipProps) {
  const { resolvedTheme } = useTheme();
  const monacoTheme = resolvedTheme === "dark" ? "vs-dark" : "vs";

  // Use wider tooltip for XML and text content
  const maxWidth = contentType === "xml" || contentType === "text" ? "max-w-[700px]" : "max-w-[500px]";

  return (
    <Tippy
      render={(attrs) => (
        <div
          {...attrs}
          className={`rounded-md border-2 border-gray-200 bg-white ${maxWidth} max-h-[400px] overflow-auto text-left shadow-lg dark:border-gray-700 dark:bg-gray-900`}
          style={{ zIndex: 10000 }}
        >
          <div className="flex items-center border-b border-gray-200 p-2 dark:border-gray-700">
            <span className="text-sm font-medium text-gray-500 dark:text-gray-400">{title}</span>
          </div>
          <div className="p-2">
            <PayloadEditor contentType={contentType} monacoTheme={monacoTheme} value={value} />
          </div>
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
