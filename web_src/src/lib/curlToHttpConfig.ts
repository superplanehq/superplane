import type { ParsedCurl } from "@/lib/parseCurl";

export type HTTPContentType =
  | "application/json"
  | "application/x-www-form-urlencoded"
  | "text/plain"
  | "application/xml";

export interface HTTPConfigurationPatch {
  method?: string;
  url?: string;
  headers?: Array<{ name: string; value: string }>;
  queryParams?: Array<{ key: string; value: string }>;
  contentType?: HTTPContentType;
  json?: unknown;
  formData?: Array<{ key: string; value: string }>;
  text?: string;
  xml?: string;
  authorization?: {
    type: "bearer" | "basic_auth" | "custom_header";
    username?: string;
    headerName?: string;
  };
}

export interface CurlToHttpConfigResult {
  patch: HTTPConfigurationPatch;
  /** Auth was detected but its credential cannot be auto-filled (secret ref required). */
  authNeedsSecret: boolean;
}

const CONTENT_TYPE_BY_HEADER: Record<string, HTTPContentType> = {
  "application/json": "application/json",
  "application/x-www-form-urlencoded": "application/x-www-form-urlencoded",
  "text/plain": "text/plain",
  "application/xml": "application/xml",
  "text/xml": "application/xml",
};

const CONTENT_TYPE_BY_FLAG: Partial<Record<NonNullable<ParsedCurl["bodyFlag"]>, HTTPContentType>> = {
  "--json": "application/json",
  "--data-urlencode": "application/x-www-form-urlencoded",
};

function inferContentType(
  contentTypeHeader: string | undefined,
  bodyFlag: ParsedCurl["bodyFlag"],
  hasBody: boolean,
): HTTPContentType | undefined {
  if (contentTypeHeader) {
    const v = contentTypeHeader.split(";")[0].trim().toLowerCase();
    if (CONTENT_TYPE_BY_HEADER[v]) return CONTENT_TYPE_BY_HEADER[v];
  }
  if (bodyFlag && CONTENT_TYPE_BY_FLAG[bodyFlag]) return CONTENT_TYPE_BY_FLAG[bodyFlag];
  return hasBody ? "application/json" : undefined;
}

function parseFormBody(body: string): Array<{ key: string; value: string }> {
  return body
    .split("&")
    .filter(Boolean)
    .map((pair) => {
      const eq = pair.indexOf("=");
      const decode = (s: string) => {
        try {
          return decodeURIComponent(s);
        } catch {
          return s;
        }
      };
      return {
        key: decode(eq < 0 ? pair : pair.slice(0, eq)),
        value: decode(eq < 0 ? "" : pair.slice(eq + 1)),
      };
    });
}

function applyBody(patch: HTTPConfigurationPatch, body: string, contentType: HTTPContentType): void {
  patch.contentType = contentType;
  switch (contentType) {
    case "application/json": {
      try {
        patch.json = JSON.parse(body);
      } catch {
        patch.contentType = "text/plain";
        patch.text = body;
      }
      return;
    }
    case "application/x-www-form-urlencoded":
      patch.formData = parseFormBody(body);
      return;
    case "text/plain":
      patch.text = body;
      return;
    case "application/xml":
      patch.xml = body;
      return;
  }
}

function authPatch(auth: NonNullable<ParsedCurl["authorization"]>): HTTPConfigurationPatch["authorization"] {
  switch (auth.type) {
    case "basic_auth":
      return { type: "basic_auth", username: auth.username };
    case "bearer":
      return { type: "bearer" };
    case "custom_header":
      return { type: "custom_header", headerName: auth.headerName };
  }
}

export function curlToHttpConfig(parsed: ParsedCurl): CurlToHttpConfigResult {
  // Every curl-owned field is explicitly initialized so that re-parsing a
  // command clears any prior prefill: anything the new curl doesn't supply
  // becomes `undefined` and gets stripped by filterVisibleConfiguration when
  // the patch is merged into the form state.
  const patch: HTTPConfigurationPatch = {
    method: parsed.method,
    url: parsed.url,
    headers: undefined,
    queryParams: undefined,
    contentType: undefined,
    json: undefined,
    formData: undefined,
    text: undefined,
    xml: undefined,
    authorization: undefined,
  };

  if (parsed.queryParams.length > 0) patch.queryParams = parsed.queryParams;

  // Split Content-Type out of the headers list so we don't duplicate it.
  const ctIdx = parsed.headers.findIndex((h) => h.name.toLowerCase() === "content-type");
  const contentTypeHeader = ctIdx >= 0 ? parsed.headers[ctIdx].value : undefined;
  const headers = ctIdx >= 0 ? parsed.headers.filter((_, i) => i !== ctIdx) : parsed.headers;
  if (headers.length > 0) patch.headers = headers;

  const hasBody = parsed.body !== undefined;
  const contentType = inferContentType(contentTypeHeader, parsed.bodyFlag, hasBody);
  if (hasBody && contentType) {
    applyBody(patch, parsed.body!, contentType);
  }

  let authNeedsSecret = false;
  if (parsed.authorization) {
    patch.authorization = authPatch(parsed.authorization);
    authNeedsSecret = true;
  }

  return { patch, authNeedsSecret };
}
