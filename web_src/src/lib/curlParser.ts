/**
 * Parses curl commands and extracts HTTP method, URL, headers, body, and parameters.
 * Supports common curl formats like:
 *  - curl -X POST https://api.example.com/users -H "Content-Type: application/json" -d '{"name":"John"}'
 *  - curl -X GET https://api.example.com/users?param1=value1
 *  - curl -H "Authorization: Bearer token" -H "Content-Type: application/json" https://api.example.com/users -d '{"name":"John"}'
 */
export interface CurlParseResult {
  method: string;
  url: string;
  headers: Array<{ name: string; value: string }>;
  body?: string;
  queryParams: Array<{ key: string; value: string }>;
  error?: string;
}

/**
 * Parses a curl command string
 * @param curlCommand The curl command to parse
 * @returns CurlParseResult with parsed components or error
 */
export function parseCurlCommand(curlCommand: string): CurlParseResult {
  const result: CurlParseResult = {
    method: "GET",
    url: "",
    headers: [],
    queryParams: [],
  };

  try {
    // Remove leading and trailing whitespace
    const command = curlCommand.replace(/\\\r?\n/g, " ").trim();

    if (!command.startsWith("curl")) {
      throw new Error("Not a valid curl command");
    }

    // Split by spaces, but respect quoted strings
    const parts = splitCurlCommand(command);

    // Extract options and arguments
    let i = 1; // Start after 'curl'
    let currentMethod = "GET";
    let currentUrl = "";
    const headers: Array<{ name: string; value: string }> = [];
    let body: string | undefined;
    let queryParams: Array<{ key: string; value: string }> = [];

    while (i < parts.length) {
      const part = parts[i];

      // Handle method specification
      if (part === "-X" || part === "--request") {
        if (i + 1 < parts.length) {
          currentMethod = stripShellQuotes(parts[i + 1]).toUpperCase();
          i += 2;
        } else {
          throw new Error("Missing method after -X flag");
        }
      } else if (part.startsWith("-X") && part.length > 2) {
        currentMethod = stripShellQuotes(part.slice(2)).toUpperCase();
        i++;
      } else if (part.startsWith("--request=")) {
        currentMethod = stripShellQuotes(part.slice("--request=".length)).toUpperCase();
        i++;
      }
      // Handle headers
      else if (part === "-H" || part === "--header") {
        if (i + 1 < parts.length) {
          const header = stripShellQuotes(parts[i + 1]);
          const colonIndex = header.indexOf(":");
          if (colonIndex !== -1) {
            const name = header.substring(0, colonIndex).trim();
            const value = header.substring(colonIndex + 1).trim();
            headers.push({ name, value });
          } else {
            throw new Error(`Invalid header format: ${header}`);
          }
          i += 2;
        } else {
          throw new Error("Missing header after -H flag");
        }
      }
      // Handle body data
      else if (part === "-d" || part === "--data" || part === "--data-raw") {
        if (i + 1 < parts.length) {
          body = stripShellQuotes(parts[i + 1]);
          i += 2;
        } else {
          throw new Error("Missing body after -d flag");
        }
      }
      // Handle POST data from file
      else if (part === "--data-binary") {
        if (i + 1 < parts.length) {
          body = stripShellQuotes(parts[i + 1]);
          i += 2;
        } else {
          throw new Error("Missing body after --data-binary flag");
        }
      } else if (part.startsWith("--data=") || part.startsWith("--data-raw=")) {
        const [, value = ""] = part.split("=", 2);
        body = stripShellQuotes(value);
        i++;
      }
      // Handle URL (last argument without flag)
      else if (!part.startsWith("-") && currentUrl === "") {
        currentUrl = stripShellQuotes(part);
        i++;
      }
      // Handle query parameters from URL
      else if (part.startsWith("-") && part.includes("?")) {
        // This handles cases like `-H "Accept: application/json?param=value"`
        // But we'll process this more carefully in the URL parsing
        i++;
      } else {
        i++;
      }
    }

    // Extract URL and query parameters if present
    if (currentUrl) {
      const [url, searchParamsStr] = currentUrl.split("?", 2);
      currentUrl = url;
      if (searchParamsStr) {
        const urlSearchParams = new URLSearchParams(searchParamsStr);
        const params: Array<{ key: string; value: string }> = [];
        for (const [key, value] of urlSearchParams.entries()) {
          params.push({ key, value });
        }
        queryParams = params;
      }
    }

    // Default to POST if body is specified but no method
    if (body && currentMethod === "GET") {
      currentMethod = "POST";
    }

    result.method = currentMethod;
    result.url = currentUrl;
    result.headers = headers;
    result.body = body;
    result.queryParams = queryParams;
  } catch (error) {
    result.error = (error as Error).message || "Failed to parse curl command";
  }

  return result;
}

/**
 * Splits a curl command while respecting quoted strings that might contain spaces
 * @param command The curl command to split
 * @returns Array of command parts
 */
function splitCurlCommand(command: string): string[] {
  const parts: string[] = [];
  let currentPart = "";
  let inQuotes = false;
  let quoteChar = "";

  for (let i = 0; i < command.length; i++) {
    const char = command[i];

    if (!inQuotes && (char === '"' || char === "'")) {
      inQuotes = true;
      quoteChar = char;
      currentPart += char;
    } else if (inQuotes && char === quoteChar) {
      inQuotes = false;
      quoteChar = "";
      currentPart += char;
    } else if (!inQuotes && char === " ") {
      if (currentPart) {
        parts.push(currentPart);
        currentPart = "";
      }
    } else {
      currentPart += char;
    }
  }

  // Add the final part if it exists
  if (currentPart) {
    parts.push(currentPart);
  }

  return parts;
}

function stripShellQuotes(value: string): string {
  const trimmed = value.trim();
  if (trimmed.length >= 2) {
    const first = trimmed[0];
    const last = trimmed[trimmed.length - 1];
    if ((first === '"' && last === '"') || (first === "'" && last === "'")) {
      return trimmed.slice(1, -1);
    }
  }
  return trimmed;
}

/**
 * Format a parsed curl result back into a curl command string
 * @param parseResult The parsed curl result
 * @returns Formatted curl command
 */
export function formatCurlCommand(parseResult: CurlParseResult): string {
  const parts = ["curl"];

  // Add method
  if (parseResult.method && parseResult.method !== "GET") {
    parts.push("-X", parseResult.method);
  }

  // Add headers
  for (const header of parseResult.headers) {
    parts.push("-H", `"${header.name}: ${header.value}"`);
  }

  // Add body if present
  if (parseResult.body) {
    parts.push("-d", `"${parseResult.body}"`);
  }

  // Add URL
  if (parseResult.url) {
    // Add query parameters
    if (parseResult.queryParams && parseResult.queryParams.length > 0) {
      let urlWithParams = parseResult.url;
      const queryParamsString = parseResult.queryParams
        .map((param) => `${encodeURIComponent(param.key)}=${encodeURIComponent(param.value)}`)
        .join("&");
      urlWithParams += `?${queryParamsString}`;
      parts.push(urlWithParams);
    } else {
      parts.push(parseResult.url);
    }
  }

  return parts.join(" ");
}
