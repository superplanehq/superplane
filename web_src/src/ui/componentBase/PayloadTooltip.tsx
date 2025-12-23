import Tippy from "@tippyjs/react/headless";
import { ReactElement } from "react";
import Editor from "@monaco-editor/react";
import "tippy.js/dist/tippy.css";

interface PayloadTooltipProps {
  children: React.ReactNode;
  title: string;
  value: any;
  contentType?: "json" | "xml" | "text";
}

export function PayloadTooltip({ children, title, value, contentType = "json" }: PayloadTooltipProps) {
  const renderContent = () => {
    // For JSON objects, use Monaco Editor
    if (contentType === "json" && typeof value === "object") {
      const stringValue = JSON.stringify(value, null, 2);

      // Calculate height based on line count (max 350px)
      const lineCount = stringValue.split("\n").length;
      const lineHeight = 19; // Monaco's default line height at 12px font
      const calculatedHeight = Math.min(Math.max(lineCount * lineHeight + 10, 100), 350);

      return (
        <div className="overflow-hidden" style={{ height: `${calculatedHeight}px`, width: "100%", minWidth: "300px" }}>
          <Editor
            height="100%"
            width="100%"
            defaultLanguage="json"
            value={stringValue}
            theme="vs"
            options={{
              readOnly: true,
              minimap: { enabled: false },
              fontSize: 12,
              lineNumbers: "off",
              wordWrap: "on",
              folding: true,
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

    // For XML, use Monaco Editor with syntax highlighting
    if (contentType === "xml") {
      const stringValue = typeof value === "string" ? value : String(value);

      // Calculate height based on line count (max 350px)
      const lineCount = stringValue.split("\n").length;
      const lineHeight = 19; // Monaco's default line height at 12px font
      const calculatedHeight = Math.min(Math.max(lineCount * lineHeight + 10, 100), 350);

      return (
        <div className="overflow-hidden" style={{ height: `${calculatedHeight}px`, width: "100%", minWidth: "300px" }}>
          <Editor
            height="100%"
            width="100%"
            defaultLanguage="xml"
            value={stringValue}
            theme="vs"
            options={{
              readOnly: true,
              minimap: { enabled: false },
              fontSize: 12,
              lineNumbers: "off",
              wordWrap: "on",
              folding: true,
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

    // For plain text, use Monaco Editor
    const stringValue = typeof value === "string" ? value : JSON.stringify(value, null, 2);

    // Calculate height based on line count (max 350px)
    const lineCount = stringValue.split("\n").length;
    const lineHeight = 19; // Monaco's default line height at 12px font
    const calculatedHeight = Math.min(Math.max(lineCount * lineHeight + 10, 100), 350);

    return (
      <div className="overflow-hidden" style={{ height: `${calculatedHeight}px`, width: "100%", minWidth: "300px" }}>
        <Editor
          height="100%"
          width="100%"
          defaultLanguage="plaintext"
          value={stringValue}
          theme="vs"
          options={{
            readOnly: true,
            minimap: { enabled: false },
            fontSize: 12,
            lineNumbers: "off",
            wordWrap: "on",
            folding: false,
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
  };

  // Use wider tooltip for XML and text content
  const maxWidth = contentType === "xml" || contentType === "text" ? "max-w-[700px]" : "max-w-[500px]";

  return (
    <Tippy
      render={() => (
        <div
          className={`bg-white border-2 border-gray-200 rounded-md ${maxWidth} max-h-[400px] overflow-auto text-left shadow-lg`}
          style={{ zIndex: 9999 }}
        >
          <div className="flex items-center border-b p-2">
            <span className="font-medium text-gray-500 text-sm">{title}</span>
          </div>
          <div className="p-2">{renderContent()}</div>
        </div>
      )}
      placement="top"
      interactive={true}
      delay={200}
      zIndex={9999}
    >
      {children as ReactElement}
    </Tippy>
  );
}
