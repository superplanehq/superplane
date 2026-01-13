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

/**
 * Safely gets a nested value from an object using a path string like "field.subfield"
 */
function getNestedValue(data: any, path: string): any {
  if (!data || !path) return undefined;

  const parts = path.split(".");
  let current = data;

  for (const part of parts) {
    if (current == null || typeof current !== "object") {
      return undefined;
    }
    current = current[part];
  }

  return current;
}

/**
 * Formats a value for display in an expression
 */
function formatValueForExpression(value: any): string {
  if (value === null) {
    return "null";
  }
  if (value === undefined) {
    return "undefined";
  }

  if (typeof value === "string") {
    // Escape quotes in strings
    const escaped = value.replace(/"/g, '\\"');
    return `"${escaped}"`;
  }

  if (typeof value === "boolean") {
    return value ? "true" : "false";
  }

  if (typeof value === "number") {
    return value.toString();
  }

  if (Array.isArray(value)) {
    return `[${value.map(formatValueForExpression).join(", ")}]`;
  }

  if (typeof value === "object") {
    return `{...}`;
  }

  return String(value);
}

/**
 * Substitutes expression values by replacing $.field patterns with actual values from the data
 * Example: $.status == "active" with data {status: "active"} becomes "active" == "active"
 */
export function substituteExpressionValues(expression: string, data: any): string {
  if (!expression || !data) {
    return expression || "";
  }

  // Match patterns like $.field or $.field.subfield
  // This regex matches $ followed by a dot and then one or more word characters or dots
  const pattern = /\$\.([a-zA-Z_][a-zA-Z0-9_.]*)/g;

  return expression.replace(pattern, (match, path) => {
    const value = getNestedValue(data, path);
    return formatValueForExpression(value);
  });
}

/**
 * Parses a value from a string representation (handles strings, numbers, booleans, null, undefined)
 */
function parseValue(valueStr: string): any {
  const trimmed = valueStr.trim();

  // Handle quoted strings
  if ((trimmed.startsWith('"') && trimmed.endsWith('"')) || (trimmed.startsWith("'") && trimmed.endsWith("'"))) {
    return trimmed.slice(1, -1).replace(/\\"/g, '"').replace(/\\'/g, "'");
  }

  // Handle booleans
  if (trimmed === "true") return true;
  if (trimmed === "false") return false;

  // Handle null/undefined
  if (trimmed === "null") return null;
  if (trimmed === "undefined") return undefined;

  // Handle numbers
  if (!isNaN(Number(trimmed)) && trimmed !== "") {
    return Number(trimmed);
  }

  return trimmed;
}

/**
 * Evaluates a simple comparison expression (e.g., "bar" == "bar", 5 > 3)
 * Returns true if the comparison is true, false otherwise
 */
function evaluateComparison(left: string, operator: string, right: string): boolean {
  const leftValue = parseValue(left);
  const rightValue = parseValue(right);

  switch (operator) {
    case "==":
      return leftValue === rightValue;
    case "!=":
      return leftValue !== rightValue;
    case ">":
      return leftValue > rightValue;
    case "<":
      return leftValue < rightValue;
    case ">=":
      return leftValue >= rightValue;
    case "<=":
      return leftValue <= rightValue;
    default:
      return false;
  }
}

/**
 * Evaluates individual comparisons in a substituted expression and returns a map
 * indicating which parts of the expression correspond to which comparison and their results.
 *
 * Returns a Set of strings representing the parts (left operand, operator, right operand)
 * that belong to comparisons that evaluated to false.
 */
export function evaluateIndividualComparisons(substitutedExpression: string): Set<string> {
  const failedParts = new Set<string>();

  if (!substitutedExpression) {
    return failedParts;
  }

  // Split by logical operators to get individual comparisons
  // We'll use a regex to split while preserving the operators
  const logicalOpRegex = /\s+(&&|\|\||and|or)\s+/gi;
  const parts = substitutedExpression.split(logicalOpRegex);

  // Process each part (alternating between comparisons and logical operators)
  for (let i = 0; i < parts.length; i += 2) {
    const comparisonPart = parts[i]?.trim();
    if (!comparisonPart) continue;

    // Try to find a comparison operator in this part
    // Order matters: check for >= and <= before > and <, and == before =
    const comparisonOpRegex = /\s*(>=|<=|==|!=|>|<|contains|startswith|endswith|matches|in)\s*/i;
    const match = comparisonPart.match(comparisonOpRegex);

    if (match && match.index !== undefined) {
      const operator = match[1];
      // Find the actual position of the operator in the string
      // match.index points to the start of the match (including leading whitespace)
      // We need to find where the operator itself starts
      const matchStart = match.index;
      const fullMatch = match[0];
      const operatorInMatch = fullMatch.indexOf(operator);
      const operatorIndex = matchStart + operatorInMatch;
      const left = comparisonPart.substring(0, operatorIndex).trim();
      const right = comparisonPart.substring(operatorIndex + operator.length).trim();

      // For now, we'll handle basic operators (==, !=, >, <, >=, <=)
      // More complex operators like "contains" would need more sophisticated parsing
      if (["==", "!=", ">", "<", ">=", "<="].includes(operator)) {
        const result = evaluateComparison(left, operator, right);

        if (!result) {
          // Mark all parts of this failed comparison
          // Store them exactly as they appear in the expression (with quotes if present)
          failedParts.add(left);
          failedParts.add(operator);
          failedParts.add(right);
        }
      }
    }
  }

  return failedParts;
}
