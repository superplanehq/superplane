import React from "react";
import Tippy from "@tippyjs/react/headless";
import "tippy.js/dist/tippy.css";

export const ExpressionTooltip: React.FC<{ expression: string; children: React.ReactElement }> = ({
  expression,
  children,
}) => {
  return (
    <Tippy
      render={() => (
        <div className="bg-white border-2 border-gray-200 rounded-md shadow-lg">
          <div className="flex items-center border-b-2 p-2">
            <span className="font-medium text-gray-500 text-sm">Expression</span>
          </div>
          <div className="p-2 max-w-xs">
            <span className="px-2 py-1 rounded-md text-sm font-mono font-medium bg-purple-100 text-purple-700 break-all">
              {expression}
            </span>
          </div>
        </div>
      )}
      placement="top"
      interactive={true}
      delay={200}
    >
      {children}
    </Tippy>
  );
};

const EXPRESSION_TOKEN_PATTERN =
  /({{|}}|\$|"(?:[^"\\]|\\.)*"|'(?:[^'\\]|\\.)*'|\b\d+(?:\.\d+)?\b|\b(?:root|previous|secrets|date|duration|now|timezone|int)\b|\b(?:true|false|null)\b|==|!=|>=|<=|&&|\|\||[?:+\-*/%().,[\]]|[A-Za-z_][A-Za-z0-9_]*|\s+)/g;

function tokenClass(token: string): string | undefined {
  if (/^\s+$/.test(token)) return undefined;
  if (token === "{{" || token === "}}") return "text-gray-500";
  if (token === "$") return "text-emerald-700";
  if (token.startsWith('"') || token.startsWith("'")) return "text-amber-700";
  if (/^\b\d+(\.\d+)?\b$/.test(token)) return "text-blue-700";
  if (/^\b(?:root|previous|secrets|date|duration|now|timezone|int)\b$/.test(token)) return "text-purple-700";
  if (/^\b(?:true|false|null)\b$/.test(token)) return "text-emerald-700";
  if (/^(==|!=|>=|<=|&&|\|\||[?:+\-*/%().,[\]])$/.test(token)) return "text-gray-600";
  return "text-gray-800 dark:text-gray-100";
}

function renderHighlightedExpression(expression: string): React.ReactNode {
  const tokens = expression.match(EXPRESSION_TOKEN_PATTERN) ?? [expression];
  return tokens.map((token, index) => {
    const className = tokenClass(token);
    if (!className) {
      return <React.Fragment key={`${token}-${index}`}>{token}</React.Fragment>;
    }
    return (
      <span key={`${token}-${index}`} className={className}>
        {token}
      </span>
    );
  });
}

export function ExpressionEnvironment() {
  return (
    <div className="space-y-1">
      <div className="font-medium text-gray-700 dark:text-gray-300">Expression environment:</div>
      <ul className="list-disc pl-4 text-gray-700 dark:text-gray-300">
        <li>
          <span className="font-mono">$</span>: run context data
        </li>
        <li>
          <span className="font-mono">root()</span>: root event data
        </li>
        <li>
          <span className="font-mono">previous()</span>: previous node outputs (optional depth)
        </li>
        <li>
          <span className="font-mono">secrets(&quot;name&quot;).key</span>: organization secret value (resolved at
          execution time)
        </li>
      </ul>
    </div>
  );
}

export function ExpressionExamples({ examples }: { examples: string[] }) {
  return (
    <div className="space-y-1">
      <div className="font-medium text-gray-700 dark:text-gray-300">Examples:</div>
      <div className="space-y-1 font-mono text-xs">
        {examples.map((example) => (
          <div key={example} className="rounded bg-gray-50 dark:bg-gray-800 px-2 py-1">
            {renderHighlightedExpression(example)}
          </div>
        ))}
      </div>
    </div>
  );
}
