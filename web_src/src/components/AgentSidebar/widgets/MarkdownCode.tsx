import type { ComponentProps, ReactNode } from "react";
import { CodeBlockWidget } from "./CodeBlockWidget";

export function MarkdownCode({ className, children, ...props }: ComponentProps<"code"> & { children?: ReactNode }) {
  const match = /language-(\w+)/.exec(className || "");
  const code = String(children).replace(/\n$/, "");

  if (match) {
    return <CodeBlockWidget code={code} language={match[1]} />;
  }

  return (
    <code className={className} {...props}>
      {children}
    </code>
  );
}
