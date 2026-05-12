export type ParsedHeader = { name: string; value: string };
export type ParsedQueryParam = { key: string; value: string };

export type ParsedAuthorization =
  | { type: "bearer"; token: string }
  | { type: "basic_auth"; username: string; password: string }
  | { type: "custom_header"; headerName: string; value: string };

export interface ParsedCurl {
  url: string;
  method: string;
  headers: ParsedHeader[];
  authorization?: ParsedAuthorization;
  body?: string;
  bodyFlag?: "-d" | "--data" | "--data-raw" | "--data-urlencode" | "--json";
  queryParams: ParsedQueryParam[];
}

const HEADERS_RE = /(?<HEADERS>-H\s+(?:'(?<hSingle>[^']*)'|"(?<hDouble>(?:\\.|[^"\\])*)"|(?<hBare>\S+:\S+)))/g;
const METHOD_RE = /(?<METHOD>-X\s+(POST|DELETE|PATCH|PUT|GET))/i;
const AUTHORIZATION_RE = /(?<AUTHORIZATION>(?<basic>-u\s+\S+:\S+)|(?<bearer>--oauth2-bearer\s+\S+))/;
const BODY_RE =
  /(?<BODY>(?<bFlag>-d|--data-urlencode|--data-raw|--data|--json)\s+(?:'(?<bSingle>[^']*)'|"(?<bDouble>(?:\\.|[^"\\])*)"))/g;
const QUERY_PARAMS_RE = /(?<QUERY_PARAMS>--url-query\s+(?:'(?<qSingle>[^']*)'|"(?<qDouble>(?:\\.|[^"\\])*)"))/g;

// curl options that consume the next argument as their value. Anything not in
// this set is treated as a boolean flag (no value), so the URL detector skips
// past it correctly. Long forms (--opt=value) are handled separately.
const OPTIONS_WITH_VALUE = new Set([
  "-X",
  "--request",
  "-H",
  "--header",
  "-u",
  "--user",
  "--oauth2-bearer",
  "-d",
  "--data",
  "--data-raw",
  "--data-binary",
  "--data-urlencode",
  "--json",
  "-F",
  "--form",
  "--url-query",
  "-A",
  "--user-agent",
  "-e",
  "--referer",
  "-b",
  "--cookie",
  "-o",
  "--output",
  "-T",
  "--upload-file",
  "--max-time",
  "--connect-timeout",
  "--retry",
  "--retry-delay",
  "--retry-max-time",
  "--cacert",
  "--cert",
  "--key",
  "--resolve",
  "-w",
  "--write-out",
]);

function unescapeDouble(s: string): string {
  return s.replace(/\\(.)/g, "$1");
}

function pick(
  groups: Record<string, string | undefined>,
  singleKey: string,
  doubleKey: string,
  bareKey?: string,
): string | undefined {
  if (groups[singleKey] !== undefined) return groups[singleKey];
  if (groups[doubleKey] !== undefined) return unescapeDouble(groups[doubleKey]!);
  if (bareKey && groups[bareKey] !== undefined) return groups[bareKey];
  return undefined;
}

function decode(s: string): string {
  try {
    return decodeURIComponent(s);
  } catch {
    return s;
  }
}

/** Split a string at the first occurrence of `sep`. Returns `[whole, ""]` when absent. */
function splitOnce(s: string, sep: string): [string, string] {
  const i = s.indexOf(sep);
  return i < 0 ? [s, ""] : [s.slice(0, i), s.slice(i + 1)];
}

function parseAuthHeaderValue(value: string): ParsedAuthorization {
  const bearer = /^Bearer\s+(.+)$/i.exec(value);
  if (bearer) return { type: "bearer", token: bearer[1].trim() };

  const basic = /^Basic\s+(.+)$/i.exec(value);
  if (basic) {
    const b64 = basic[1].trim();
    try {
      const decoded = typeof atob === "function" ? atob(b64) : Buffer.from(b64, "base64").toString("utf8");
      const [username, password] = splitOnce(decoded, ":");
      return { type: "basic_auth", username, password };
    } catch {
      // fall through to custom-header treatment
    }
  }

  return { type: "custom_header", headerName: "Authorization", value };
}

/**
 * Shell-style tokenizer: splits on whitespace, respects single quotes (literal),
 * double quotes (backslash-escaped), and lets adjacent quoted/unquoted segments
 * concatenate into one token (e.g. `--data="hello"` → `--data=hello`).
 */
function readSingleQuoted(src: string, i: number): [string, number] {
  let out = "";
  i++; // skip opening quote
  while (i < src.length && src[i] !== "'") out += src[i++];
  if (i < src.length) i++; // skip closing quote
  return [out, i];
}

function readDoubleQuoted(src: string, i: number): [string, number] {
  let out = "";
  i++; // skip opening quote
  while (i < src.length && src[i] !== '"') {
    const esc = src[i] === "\\" && i + 1 < src.length;
    out += esc ? src[i + 1] : src[i];
    i += esc ? 2 : 1;
  }
  if (i < src.length) i++; // skip closing quote
  return [out, i];
}

function readToken(src: string, i: number): [string, number] {
  let out = "";
  while (i < src.length && !/\s/.test(src[i])) {
    const c = src[i];
    if (c === "'") {
      const [chunk, next] = readSingleQuoted(src, i);
      out += chunk;
      i = next;
    } else if (c === '"') {
      const [chunk, next] = readDoubleQuoted(src, i);
      out += chunk;
      i = next;
    } else {
      out += src[i++];
    }
  }
  return [out, i];
}

function tokenize(src: string): string[] {
  const tokens: string[] = [];
  let i = 0;
  while (i < src.length) {
    while (i < src.length && /\s/.test(src[i])) i++;
    if (i >= src.length) break;
    const [token, next] = readToken(src, i);
    tokens.push(token);
    i = next;
  }
  return tokens;
}

/**
 * Find the URL using curl's own argument-parsing rule: anything starting with
 * `-`/`--` is an option (and may consume the next token), the first remaining
 * positional argument is the URL. Works for bare hostnames, IPs, localhost
 * with ports, custom schemes, etc. — not just http(s)://.
 */
function extractUrl(src: string): { url: string; inlineQuery: string } {
  const tokens = tokenize(src);
  let i = tokens[0] === "curl" ? 1 : 0;

  while (i < tokens.length) {
    const tok = tokens[i];

    if (tok.startsWith("-")) {
      // --opt=value form consumes its own value
      if (tok.startsWith("--") && tok.includes("=")) {
        i++;
        continue;
      }
      // Known option that takes a value → also skip the next token
      i += OPTIONS_WITH_VALUE.has(tok) ? 2 : 1;
      continue;
    }

    const [url, inlineQuery] = splitOnce(tok, "?");
    return { url, inlineQuery };
  }

  return { url: "", inlineQuery: "" };
}

function extractHeaders(src: string): ParsedHeader[] {
  const headers: ParsedHeader[] = [];
  for (const m of src.matchAll(HEADERS_RE)) {
    const raw = pick(m.groups ?? {}, "hSingle", "hDouble", "hBare");
    if (!raw) continue;
    const sep = raw.indexOf(":");
    if (sep <= 0) continue;
    headers.push({ name: raw.slice(0, sep).trim(), value: raw.slice(sep + 1).trim() });
  }
  return headers;
}

function extractAuthorization(src: string, headers: ParsedHeader[]): ParsedAuthorization | undefined {
  const m = src.match(AUTHORIZATION_RE);
  if (m?.groups?.basic) {
    const creds = m.groups.basic.replace(/^-u\s+/, "");
    const [username, password] = splitOnce(creds, ":");
    return { type: "basic_auth", username, password };
  }
  if (m?.groups?.bearer) {
    return { type: "bearer", token: m.groups.bearer.replace(/^--oauth2-bearer\s+/, "") };
  }

  const i = headers.findIndex((h) => h.name.toLowerCase() === "authorization");
  if (i < 0) return undefined;
  const [removed] = headers.splice(i, 1);
  return parseAuthHeaderValue(removed.value);
}

function extractBody(src: string): { body?: string; bodyFlag?: ParsedCurl["bodyFlag"] } {
  let body: string | undefined;
  let bodyFlag: ParsedCurl["bodyFlag"];
  const urlencodePairs: string[] = [];

  for (const m of src.matchAll(BODY_RE)) {
    const flag = m.groups?.bFlag as ParsedCurl["bodyFlag"];
    const value = pick(m.groups ?? {}, "bSingle", "bDouble");
    if (value === undefined) continue;

    if (flag === "--data-urlencode") {
      urlencodePairs.push(value);
      bodyFlag = bodyFlag ?? flag;
      continue;
    }
    if (body === undefined) {
      body = value;
      bodyFlag = flag;
    }
  }

  if (urlencodePairs.length > 0 && bodyFlag === "--data-urlencode") {
    body = urlencodePairs.join("&");
  }
  return { body, bodyFlag };
}

function extractQueryParams(src: string, inlineQuery: string): ParsedQueryParam[] {
  const out: ParsedQueryParam[] = [];

  if (inlineQuery) {
    for (const pair of inlineQuery.split("&")) {
      if (!pair) continue;
      const [k, v] = splitOnce(pair, "=");
      out.push({ key: decode(k), value: decode(v) });
    }
  }

  for (const m of src.matchAll(QUERY_PARAMS_RE)) {
    const pair = pick(m.groups ?? {}, "qSingle", "qDouble");
    if (!pair) continue;
    const [key, value] = splitOnce(pair, "=");
    out.push({ key, value });
  }

  return out;
}

export function parseCurl(curlCmd: string): ParsedCurl {
  const src = curlCmd.replace(/\\\r?\n/g, " ").trim();

  const { url, inlineQuery } = extractUrl(src);
  const methodMatch = src.match(METHOD_RE);
  const headers = extractHeaders(src);
  const authorization = extractAuthorization(src, headers);
  const { body, bodyFlag } = extractBody(src);
  const queryParams = extractQueryParams(src, inlineQuery);

  const method = methodMatch ? methodMatch[2].toUpperCase() : body !== undefined ? "POST" : "GET";

  return {
    url,
    method,
    headers,
    ...(authorization && { authorization }),
    ...(body !== undefined && { body, bodyFlag }),
    queryParams,
  };
}
