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
  const splittedExpression = splitExpressionBySpaces(expression);
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

const splitExpressionBySpaces = (input: string): string[] => {
  const parts: string[] = [];
  let current = "";
  let inSingle = false;
  let inDouble = false;
  let parenDepth = 0;
  let bracketDepth = 0;

  const isEscaped = (idx: number) => {
    let backslashes = 0;
    for (let i = idx - 1; i >= 0 && input[i] === "\\"; i--) {
      backslashes++;
    }
    return backslashes % 2 === 1;
  };

  for (let i = 0; i < input.length; i++) {
    const ch = input[i];

    if (!inDouble && ch === "'" && !isEscaped(i)) {
      inSingle = !inSingle;
      current += ch;
      continue;
    }
    if (!inSingle && ch === '"' && !isEscaped(i)) {
      inDouble = !inDouble;
      current += ch;
      continue;
    }

    if (!inSingle && !inDouble) {
      if (ch === "(") parenDepth++;
      if (ch === ")") parenDepth = Math.max(0, parenDepth - 1);
      if (ch === "[") bracketDepth++;
      if (ch === "]") bracketDepth = Math.max(0, bracketDepth - 1);
    }

    if (!inSingle && !inDouble && parenDepth === 0 && bracketDepth === 0 && /\s/.test(ch)) {
      if (current) {
        parts.push(current);
        current = "";
      }
      continue;
    }

    current += ch;
  }

  if (current) {
    parts.push(current);
  }

  return parts;
};

function getNestedValueByTokens(data: any, tokens: Array<string | number>): any {
  if (!data) return undefined;
  let current = data;
  for (const token of tokens) {
    if (current == null || typeof current !== "object") {
      return undefined;
    }
    current = (current as any)[token];
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
type SubstituteContext = {
  root?: any;
  previousByDepth?: Record<string, any>;
};

export function substituteExpressionValues(expression: string, data: any, context?: SubstituteContext): string {
  if (!expression || !data) {
    return expression || "";
  }

  const isEscaped = (input: string, index: number) => {
    let backslashes = 0;
    for (let i = index - 1; i >= 0 && input[i] === "\\"; i--) {
      backslashes++;
    }
    return backslashes % 2 === 1;
  };

  const parseExpressionReference = (input: string, startIndex: number) => {
    if (input[startIndex] !== "$") return null;

    let i = startIndex + 1;
    const tokens: Array<string | number> = [];

    while (i < input.length) {
      const ch = input[i];
      if (ch === ".") {
        i++;
        const identMatch = input.slice(i).match(/^[$A-Za-z_][$A-Za-z0-9_]*/);
        if (!identMatch) break;
        tokens.push(identMatch[0]);
        i += identMatch[0].length;
        continue;
      }

      if (ch === "[") {
        i++;
        while (i < input.length && /\s/.test(input[i])) i++;
        if (i >= input.length) break;

        const quote = input[i] === "'" || input[i] === '"' ? input[i] : null;
        if (quote) {
          i++;
          let value = "";
          while (i < input.length && (input[i] !== quote || isEscaped(input, i))) {
            value += input[i];
            i++;
          }
          if (input[i] !== quote) break;
          i++;
          while (i < input.length && /\s/.test(input[i])) i++;
          if (input[i] !== "]") break;
          i++;
          tokens.push(value.replace(/\\(["'\\])/g, "$1"));
          continue;
        }

        const numberMatch = input.slice(i).match(/^\d+/);
        if (numberMatch) {
          i += numberMatch[0].length;
          while (i < input.length && /\s/.test(input[i])) i++;
          if (input[i] !== "]") break;
          i++;
          tokens.push(Number(numberMatch[0]));
          continue;
        }

        break;
      }

      break;
    }

    return { endIndex: i, tokens };
  };

  const parseFunctionReference = (input: string, startIndex: number) => {
    if (startIndex > 0 && /[$A-Za-z0-9_]/.test(input[startIndex - 1])) {
      return null;
    }

    const remaining = input.slice(startIndex);
    let name = "";
    let i = startIndex;

    if (remaining.startsWith("root")) {
      name = "root";
      i += 4;
    } else if (remaining.startsWith("previous")) {
      name = "previous";
      i += 8;
    } else {
      return null;
    }

    while (i < input.length && /\s/.test(input[i])) i++;
    if (input[i] !== "(") return null;
    i++;

    let depthValue = "1";
    if (name === "previous") {
      let args = "";
      let parenDepth = 1;
      while (i < input.length) {
        const ch = input[i];
        if (ch === "(") parenDepth++;
        if (ch === ")") {
          parenDepth--;
          if (parenDepth === 0) {
            i++;
            break;
          }
        }
        if (parenDepth > 0) {
          args += ch;
        }
        i++;
      }

      const trimmed = args.trim();
      if (trimmed !== "") {
        depthValue = trimmed;
      }
    } else {
      while (i < input.length && /\s/.test(input[i])) i++;
      if (input[i] !== ")") return null;
      i++;
    }

    const tokens: Array<string | number> = [];
    while (i < input.length) {
      const ch = input[i];
      if (ch === ".") {
        i++;
        const identMatch = input.slice(i).match(/^[$A-Za-z_][$A-Za-z0-9_]*/);
        if (!identMatch) break;
        tokens.push(identMatch[0]);
        i += identMatch[0].length;
        continue;
      }

      if (ch === "[") {
        i++;
        while (i < input.length && /\s/.test(input[i])) i++;
        if (i >= input.length) break;

        const quote = input[i] === "'" || input[i] === '"' ? input[i] : null;
        if (quote) {
          i++;
          let value = "";
          while (i < input.length && (input[i] !== quote || isEscaped(input, i))) {
            value += input[i];
            i++;
          }
          if (input[i] !== quote) break;
          i++;
          while (i < input.length && /\s/.test(input[i])) i++;
          if (input[i] !== "]") break;
          i++;
          tokens.push(value.replace(/\\(["'\\])/g, "$1"));
          continue;
        }

        const numberMatch = input.slice(i).match(/^\d+/);
        if (numberMatch) {
          i += numberMatch[0].length;
          while (i < input.length && /\s/.test(input[i])) i++;
          if (input[i] !== "]") break;
          i++;
          tokens.push(Number(numberMatch[0]));
          continue;
        }

        break;
      }

      break;
    }

    return { endIndex: i, tokens, name, depthValue };
  };

  let out = "";
  let inSingle = false;
  let inDouble = false;
  const rootPayload = context?.root ?? data?.__root;
  const previousByDepth = context?.previousByDepth ?? data?.__previousByDepth;

  for (let i = 0; i < expression.length; i++) {
    const ch = expression[i];
    if (!inDouble && ch === "'" && !isEscaped(expression, i)) {
      inSingle = !inSingle;
      out += ch;
      continue;
    }
    if (!inSingle && ch === '"' && !isEscaped(expression, i)) {
      inDouble = !inDouble;
      out += ch;
      continue;
    }

    if (!inSingle && !inDouble && !isEscaped(expression, i)) {
      if (ch === "$") {
        const parsed = parseExpressionReference(expression, i);
        if (parsed) {
          let value = parsed.tokens.length === 0 ? data : getNestedValueByTokens(data, parsed.tokens);
          if (
            value === undefined &&
            parsed.tokens.length > 1 &&
            typeof parsed.tokens[0] === "string" &&
            !Object.prototype.hasOwnProperty.call(data, parsed.tokens[0])
          ) {
            value = getNestedValueByTokens(data, parsed.tokens.slice(1));
          }
          out += formatValueForExpression(value);
          i = parsed.endIndex - 1;
          continue;
        }
      }

      const parsedFn = parseFunctionReference(expression, i);
      if (parsedFn) {
        let baseValue: any = undefined;
        if (parsedFn.name === "root") {
          baseValue = rootPayload;
        } else if (parsedFn.name === "previous") {
          baseValue = previousByDepth?.[parsedFn.depthValue] ?? undefined;
        }

        const value = parsedFn.tokens.length === 0 ? baseValue : getNestedValueByTokens(baseValue, parsedFn.tokens);
        out += formatValueForExpression(value);
        i = parsedFn.endIndex - 1;
        continue;
      }
    }

    out += ch;
  }

  return out;
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
