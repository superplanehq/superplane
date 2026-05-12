export type ParsedHeader = { name: string; value: string };
export type ParsedQueryParam = { key: string; value: string };

export interface ParsedCurlRequest {
  method: string;
  url: string;
  headers?: ParsedHeader[];
  queryParams?: ParsedQueryParam[];
  contentType?: string;
  json?: unknown;
  formData?: ParsedQueryParam[];
  text?: string;
}

export interface ParseCurlResult {
  success: boolean;
  request?: ParsedCurlRequest;
  warnings: string[];
  errors: string[];
}

const UNSUPPORTED_FLAGS = new Set([
  "-u",
  "--user",
  "--cert",
  "--key",
  "-k",
  "--insecure",
  "--proxy",
  "--proxy-user",
]);

const SKIPPED_FLAGS = new Set(["--compressed", "-L", "--location", "--silent", "-s", "--verbose", "-v"]);

function isString(value: unknown): value is string {
  return typeof value === "string";
}

function stripLineContinuations(input: string): string {
  return input.replace(/\\\r?\n/g, " ");
}

function tokenizeShellLike(input: string): string[] {
  const tokens: string[] = [];
  let current = "";
  let quote: "'" | '"' | null = null;
  let escaping = false;
  let justClosedQuote = false;

  for (let index = 0; index < input.length; index += 1) {
    const char = input[index];

    if (escaping) {
      current += char;
      escaping = false;
      continue;
    }

    if (char === "\\") {
      escaping = true;
      continue;
    }

    if (quote) {
      if (char === quote) {
        quote = null;
        justClosedQuote = true;
      } else {
        current += char;
      }
      continue;
    }

    if (char === "'" || char === '"') {
      quote = char;
      continue;
    }

    if (/\s/.test(char)) {
      if (current.length > 0 || justClosedQuote) {
        tokens.push(current);
        current = "";
        justClosedQuote = false;
      }
      continue;
    }

    justClosedQuote = false;
    current += char;
  }

  if (escaping) {
    current += "\\";
  }

  if (current.length > 0 || justClosedQuote) {
    tokens.push(current);
  }

  return tokens;
}

function maybeParseJson(value: string): { json?: unknown; text?: string; warning?: string } {
  const trimmed = value.trim();
  if (trimmed.length === 0) {
    return { text: "" };
  }

  if (!trimmed.startsWith("{") && !trimmed.startsWith("[") && !trimmed.startsWith('"')) {
    return { text: value };
  }

  try {
    return { json: JSON.parse(trimmed) };
  } catch {
    return { text: value, warning: "Malformed JSON body parsed as plain text" };
  }
}

function parseHeader(rawHeader: string): ParsedHeader | null {
  const separatorIndex = rawHeader.indexOf(":");
  if (separatorIndex < 0) {
    return null;
  }

  const name = rawHeader.slice(0, separatorIndex).trim();
  const value = rawHeader.slice(separatorIndex + 1).trim();
  if (!name) {
    return null;
  }

  return { name, value };
}

function extractQueryParams(rawUrl: string): ParsedQueryParam[] {
  try {
    const parsed = new URL(rawUrl);
    const params: ParsedQueryParam[] = [];
    parsed.searchParams.forEach((value, key) => {
      params.push({ key, value });
    });
    return params;
  } catch {
    return [];
  }
}

function inferMethod(explicitMethod: string | undefined, hasBody: boolean): string {
  if (explicitMethod && explicitMethod.trim().length > 0) {
    return explicitMethod.toUpperCase();
  }

  return hasBody ? "POST" : "GET";
}

function inferContentTypeFromHeaders(headers: ParsedHeader[]): string | undefined {
  const match = headers.find((header) => header.name.toLowerCase() === "content-type");
  return match?.value;
}

export function parseCurl(input: unknown): ParseCurlResult {
  if (!isString(input) || input.trim().length === 0) {
    return { success: false, warnings: [], errors: ["Input is required"] };
  }

  const normalized = stripLineContinuations(input).trim();
  const tokens = tokenizeShellLike(normalized);

  if (tokens.length === 0 || tokens[0] !== "curl") {
    return { success: false, warnings: [], errors: ["Invalid curl command"] };
  }

  let method: string | undefined;
  let url: string | undefined;
  let bodyText: string | undefined;
  let sawBodyFlag = false;
  let sawDataUrlEncoded = false;
  const headers: ParsedHeader[] = [];
  const formData: ParsedQueryParam[] = [];
  const warnings: string[] = [];

  for (let index = 1; index < tokens.length; index += 1) {
    const token = tokens[index];

    if (SKIPPED_FLAGS.has(token)) {
      continue;
    }

    if (UNSUPPORTED_FLAGS.has(token)) {
      warnings.push(`Unsupported flag ignored: ${token}`);
      if (index + 1 < tokens.length && !tokens[index + 1].startsWith("-")) {
        index += 1;
      }
      continue;
    }

    if ((token === "-X" || token === "--request") && tokens[index + 1]) {
      method = tokens[index + 1];
      index += 1;
      continue;
    }

    if ((token === "-H" || token === "--header") && tokens[index + 1]) {
      const parsed = parseHeader(tokens[index + 1]);
      if (parsed) {
        headers.push(parsed);
      } else {
        warnings.push(`Invalid header ignored: ${tokens[index + 1]}`);
      }
      index += 1;
      continue;
    }

    if (
      (token === "-d" || token === "--data" || token === "--data-raw" || token === "--data-binary") &&
      tokens[index + 1] !== undefined
    ) {
      sawBodyFlag = true;
      bodyText = tokens[index + 1];
      index += 1;
      continue;
    }

    if (token === "--data-urlencode" && tokens[index + 1] !== undefined) {
      sawBodyFlag = true;
      sawDataUrlEncoded = true;
      const value = tokens[index + 1];
      const separatorIndex = value.indexOf("=");
      if (separatorIndex >= 0) {
        formData.push({
          key: value.slice(0, separatorIndex),
          value: value.slice(separatorIndex + 1),
        });
      } else {
        formData.push({ key: value, value: "" });
      }
      index += 1;
      continue;
    }

    if ((token === "-F" || token === "--form") && tokens[index + 1] !== undefined) {
      const value = tokens[index + 1];
      const separatorIndex = value.indexOf("=");
      if (separatorIndex >= 0) {
        formData.push({
          key: value.slice(0, separatorIndex),
          value: value.slice(separatorIndex + 1),
        });
      } else {
        formData.push({ key: value, value: "" });
      }
      warnings.push(`Partially supported flag parsed as form data: ${token}`);
      sawBodyFlag = true;
      index += 1;
      continue;
    }

    if (!token.startsWith("-") && !url) {
      url = token;
      continue;
    }
  }

  if (!url) {
    return { success: false, warnings, errors: ["Invalid curl command"] };
  }

  const request: ParsedCurlRequest = {
    method: inferMethod(method, sawBodyFlag),
    url,
  };

  if (headers.length > 0) {
    request.headers = headers;
  }

  const extractedQueryParams = extractQueryParams(url);
  if (extractedQueryParams.length > 0) {
    request.queryParams = extractedQueryParams;
  }

  if (formData.length > 0 || sawDataUrlEncoded) {
    request.contentType = "application/x-www-form-urlencoded";
    request.formData = formData;
  } else if (sawBodyFlag) {
    const parseResult = maybeParseJson(bodyText ?? "");
    if (parseResult.warning) {
      warnings.push(parseResult.warning);
    }

    const contentTypeHeader = inferContentTypeFromHeaders(headers);
    request.contentType = contentTypeHeader || (parseResult.json !== undefined ? "application/json" : "text/plain");
    if (parseResult.json !== undefined) {
      request.json = parseResult.json;
    } else {
      request.text = parseResult.text ?? "";
    }
  }

  return {
    success: true,
    request,
    warnings,
    errors: [],
  };
}
