import { useMemo } from "react";

import { highlightCode, type CodeLanguage, type CodeTokenKind } from "@/lib/codeHighlight";
import { cn } from "@/lib/utils";

const TOKEN_CLASS: Record<CodeTokenKind, string | undefined> = {
  plain: undefined,
  key: "text-indigo-700 dark:text-indigo-300",
  string: "text-amber-700 dark:text-amber-300",
  number: "text-emerald-700 dark:text-emerald-300",
  literal: "text-blue-700 dark:text-blue-300",
  punctuation: "text-muted-foreground",
  command: "text-purple-700 dark:text-purple-300",
  flag: "text-sky-700 dark:text-sky-300",
  variable: "text-pink-700 dark:text-pink-300",
  comment: "text-muted-foreground italic",
};

export function HighlightedCode({
  code,
  language,
  className,
  testId,
}: {
  code: string;
  language: CodeLanguage;
  className?: string;
  testId?: string;
}) {
  const tokens = useMemo(() => highlightCode(code, language), [code, language]);
  return (
    <pre className={cn("overflow-x-auto text-[12px] leading-relaxed text-foreground", className)} data-testid={testId}>
      <code>
        {tokens.map((token, index) => (
          <span key={index} className={TOKEN_CLASS[token.kind]}>
            {token.text}
          </span>
        ))}
      </code>
    </pre>
  );
}
