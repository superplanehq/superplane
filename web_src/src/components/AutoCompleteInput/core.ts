/* eslint-disable @typescript-eslint/no-explicit-any */

/**
 * Get the value at a specific path in an object
 * @param {Object} obj - The object to traverse
 * @param {string} path - The path to the value (e.g., "test.my_name" or "items[0].name")
 * @returns {any} The value at the path, or undefined if not found
 */
export function getValueAtPath(obj: any, path: string): any {
  if (!obj || !path) return undefined;

  try {
    const parts = parsePathSegments(path);
    let current = obj;
    for (const part of parts) {
      if (current === null || current === undefined) {
        return undefined;
      }
      current = current[part as keyof typeof current];
    }

    return current;
  } catch {
    return undefined;
  }
}

export function parsePathSegments(path: string): Array<string | number> {
  const segments: Array<string | number> = [];
  if (!path) {
    return segments;
  }

  let i = 0;
  if (path[i] === "$") {
    i += 1;
    if (path[i] === ".") {
      i += 1;
    }
  }

  while (i < path.length) {
    const char = path[i];

    if (char === ".") {
      i += 1;
      continue;
    }

    if (char === "[") {
      i += 1;
      if (i >= path.length) break;

      const quote = path[i];
      if (quote === "'" || quote === '"') {
        i += 1;
        let value = "";
        while (i < path.length) {
          const nextChar = path[i];
          if (nextChar === "\\" && i + 1 < path.length) {
            value += path[i + 1];
            i += 2;
            continue;
          }
          if (nextChar === quote) {
            i += 1;
            break;
          }
          value += nextChar;
          i += 1;
        }
        while (i < path.length && path[i] !== "]") {
          i += 1;
        }
        if (path[i] === "]") {
          i += 1;
        }
        segments.push(value);
        continue;
      }

      let numberValue = "";
      while (i < path.length && /\d/.test(path[i])) {
        numberValue += path[i];
        i += 1;
      }
      while (i < path.length && path[i] !== "]") {
        i += 1;
      }
      if (path[i] === "]") {
        i += 1;
      }
      if (numberValue !== "") {
        segments.push(Number(numberValue));
      }
      continue;
    }

    let value = "";
    while (i < path.length && path[i] !== "." && path[i] !== "[") {
      value += path[i];
      i += 1;
    }
    if (value !== "") {
      segments.push(value);
    }
  }

  return segments;
}

export function buildLookupPath(segments: Array<string | number>): string {
  let path = "";
  segments.forEach((segment) => {
    if (typeof segment === "number") {
      path += `[${segment}]`;
      return;
    }

    path = path ? `${path}.${segment}` : segment;
  });

  return path;
}

export function isValidIdentifier(value: string): boolean {
  return /^[A-Za-z_][A-Za-z0-9_]*$/.test(value);
}

export function formatDisplayPath(segments: Array<string | number>, includeDollar = false): string {
  let path = includeDollar ? "$" : "";
  segments.forEach((segment) => {
    if (typeof segment === "number") {
      path += `[${segment}]`;
      return;
    }

    if (isValidIdentifier(segment)) {
      path += path ? `.${segment}` : segment;
      return;
    }

    path += `["${segment}"]`;
  });

  return path;
}

/**
 * Get the type of a value as a user-friendly string
 * @param {any} value - The value to get the type of
 * @returns {string} The type as a string
 */
export function getTypeString(value: any): string {
  if (value === null) return "null";
  if (value === undefined) return "undefined";
  if (Array.isArray(value)) return "array";

  const type = typeof value;
  if (type === "object") {
    return "object";
  }

  return type;
}

/**
 * Flattens a JSON object by parent field and depth layer for autocomplete
 * @param {Object} obj - The JSON object to flatten
 * @returns {Object} Flattened structure with parent-depth as keys and field arrays as values
 */
export function flattenForAutocomplete(obj: any) {
  const result: any = {};

  // Add root level keys
  if (typeof obj === "object" && !Array.isArray(obj)) {
    result["root-0"] = Object.keys(obj);
  }

  function traverse(current: any, parentKey: string | null = null, depth = 0) {
    if (current === null || current === undefined) {
      return;
    }

    // Handle arrays
    if (Array.isArray(current)) {
      current.forEach((item, index) => {
        const arrayKey = `${parentKey}[${index}]`;

        // Add the array index accessor to parent's suggestions
        if (parentKey) {
          const parentDepthKey = `${parentKey}-${depth}`;
          if (!result[parentDepthKey]) {
            result[parentDepthKey] = [];
          }
          if (!result[parentDepthKey].includes(arrayKey)) {
            result[parentDepthKey].push(arrayKey);
          }
        }

        // Traverse into array item
        traverse(item, arrayKey, depth + 1);
      });
      return;
    }

    // Handle objects
    if (typeof current === "object") {
      const keys = Object.keys(current);

      if (parentKey !== null) {
        const depthKey = `${parentKey}-${depth}`;
        if (!result[depthKey]) {
          result[depthKey] = [];
        }
        // Add all direct child keys
        keys.forEach((key) => {
          if (!result[depthKey].includes(key)) {
            result[depthKey].push(key);
          }
        });
      }

      // Traverse into each property
      keys.forEach((key) => {
        const newParent = parentKey ? `${parentKey}.${key}` : key;
        traverse(current[key], newParent, parentKey === null ? 0 : depth + 1);
      });
      return;
    }

    // Primitive values (string, number, boolean) - no further traversal needed
  }

  traverse(obj);
  return result;
}

/**
 * Get autocomplete suggestions based on current input path
 * @param {Object} flattenedData - The flattened data structure
 * @param {string} currentPath - The current input path (e.g., "test.myArray[0]")
 * @returns {Array} Array of suggestion strings
 */
export function getAutocompleteSuggestions(flattenedData: any, currentPath: string) {
  if (!currentPath) {
    const topLevelKeys = Object.keys(flattenedData).filter((key) => key.endsWith("-0"));

    if (topLevelKeys.length > 0) {
      return topLevelKeys.map((topLevelKey) => {
        const splittedTopLevelKey = topLevelKey.split("-");
        const keyWords = splittedTopLevelKey.slice(0, splittedTopLevelKey.length - 1);
        return keyWords.join("-");
      });
    }

    return [];
  }

  const depth = (currentPath.match(/\./g) || []).length + (currentPath.match(/\[/g) || []).length;

  const lookupKey = `${currentPath}-${depth}`;
  return flattenedData[lookupKey] || [];
}

/**
 * Get autocomplete suggestions with type information
 * @param {Object} flattenedData - The flattened data structure
 * @param {string} currentPath - The current input path
 * @param {string} basePath - The base path to build full paths from
 * @param {Object} exampleObj - The original object to get types from
 * @returns {Array} Array of objects with suggestion and type
 */
export function getAutocompleteSuggestionsWithTypes(
  flattenedData: any,
  currentPath: string,
  basePath: string,
  exampleObj: any,
): Array<{ suggestion: string; type: string }> {
  const suggestions = getAutocompleteSuggestions(flattenedData, currentPath);

  return suggestions.map((suggestion: string) => {
    // Build the full path for type checking
    let fullPath: string;
    if (currentPath === "root") {
      fullPath = suggestion;
    } else {
      // Check if suggestion is an array index or regular property
      if (suggestion.match(/\[/)) {
        fullPath = suggestion.startsWith(currentPath) ? suggestion : `${currentPath}.${suggestion}`;
      } else {
        fullPath = basePath ? `${basePath}.${suggestion}` : suggestion;
      }
    }

    const value = getValueAtPath(exampleObj, fullPath);
    const type = getTypeString(value);

    return {
      suggestion,
      type,
    };
  });
}
