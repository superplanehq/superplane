import { splitBySpaces } from "@/lib/utils";
import { ComponentBaseSpecValue } from "../ui/componentBase";

const operators = new Set([
  ">=",
  "<=",
  "==",
  "!=",
  ">",
  "<",
  "contains",
  "startswith",
  "endswith",
  "matches",
  "in",
  "!",
  "+",
  "-",
  "*",
  "/",
  "%",
  "**",
  "??",
  "?",
  ":",
]);

const logicalOperators = new Set(["and", "or", "||", "&&"]);

const isStaticValue = (value: string) => {
  if (value === "true" || value === "false") return true;
  if (value === "null" || value === "undefined") return true;
  if (value.startsWith("'") && value.endsWith("'")) return true;
  if (value.startsWith('"') && value.endsWith('"')) return true;
  if (!isNaN(Number(value))) return true;

  return false;
};

export const parseExpression = (expression: string): ComponentBaseSpecValue[] => {
  if (!expression) return [];

  const result: ComponentBaseSpecValue[] = [];
  const splittedExpression = splitBySpaces(expression);
  let current: ComponentBaseSpecValue = {
    badges: [],
  };

  for (const term of splittedExpression) {
    const normalizedTerm = term.trim().toLowerCase();
    if (operators.has(normalizedTerm)) {
      current.badges.push({ label: term, bgColor: "bg-gray-100", textColor: "text-gray-700" });
    } else if (logicalOperators.has(normalizedTerm)) {
      current.badges.push({ label: term, bgColor: "bg-gray-500", textColor: "text-white" });
      result.push(current);
      current = {
        badges: [],
      };
    } else if (isStaticValue(normalizedTerm)) {
      current.badges.push({ label: term, bgColor: "bg-green-100", textColor: "text-green-700" });
    } else {
      current.badges.push({ label: term, bgColor: "bg-purple-100", textColor: "text-purple-700" });
    }
  }

  result.push(current);
  return result;
};
