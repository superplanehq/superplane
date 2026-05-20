import { describe, expect, it } from "vitest";
import { parseCurl } from "@/lib/parseCurl";
import { curlToHttpConfig } from "@/lib/curlToHttpConfig";

const convert = (cmd: string) => curlToHttpConfig(parseCurl(cmd));

describe("curlToHttpConfig", () => {
  it("maps method, url, headers, and queryParams 1:1", () => {
    const { patch } = convert(`curl -X POST 'https://api.example.com/x?a=1&b=2' -H 'X-Trace-Id: abc'`);
    expect(patch.method).toBe("POST");
    expect(patch.url).toBe("https://api.example.com/x");
    expect(patch.queryParams).toEqual([
      { key: "a", value: "1" },
      { key: "b", value: "2" },
    ]);
    expect(patch.headers).toEqual([{ name: "X-Trace-Id", value: "abc" }]);
  });

  it("infers application/json and parses the body into `json`", () => {
    const { patch } = convert(
      `curl -X POST https://api.example.com -H 'Content-Type: application/json' -d '{"sku":"A1","qty":2}'`,
    );
    expect(patch.contentType).toBe("application/json");
    expect(patch.json).toEqual({ sku: "A1", qty: 2 });
    // Content-Type is consumed and not duplicated in headers
    expect(patch.headers ?? []).toEqual([]);
  });

  it("falls back to text/plain when body claims JSON but is invalid", () => {
    const { patch } = convert(`curl -X POST https://api.example.com -H 'Content-Type: application/json' -d 'not json'`);
    expect(patch.contentType).toBe("text/plain");
    expect(patch.text).toBe("not json");
    expect(patch.json).toBeUndefined();
  });

  it("infers form-urlencoded from --data-urlencode and splits into formData", () => {
    const { patch } = convert(
      `curl -X POST https://api.example.com --data-urlencode 'user=jane doe' --data-urlencode 'pass=hunter2'`,
    );
    expect(patch.contentType).toBe("application/x-www-form-urlencoded");
    expect(patch.formData).toEqual([
      { key: "user", value: "jane doe" },
      { key: "pass", value: "hunter2" },
    ]);
  });

  it("infers application/json from --json shortcut", () => {
    const { patch } = convert(`curl --json '{"a":1}' https://api.example.com`);
    expect(patch.contentType).toBe("application/json");
    expect(patch.json).toEqual({ a: 1 });
  });

  it("routes text/plain bodies into `text`", () => {
    const { patch } = convert(
      `curl -X POST https://api.example.com -H 'Content-Type: text/plain' --data-raw 'hello world'`,
    );
    expect(patch.contentType).toBe("text/plain");
    expect(patch.text).toBe("hello world");
  });

  it("routes application/xml bodies into `xml`", () => {
    const { patch } = convert(
      `curl -X POST https://api.example.com -H 'Content-Type: application/xml' --data-raw '<root/>'`,
    );
    expect(patch.contentType).toBe("application/xml");
    expect(patch.xml).toBe("<root/>");
  });

  it("translates basic auth (username only, secret left blank)", () => {
    const { patch, authNeedsSecret } = convert(`curl -u alice:s3cret https://api.example.com`);
    expect(patch.authorization).toEqual({ type: "basic_auth", username: "alice" });
    expect(authNeedsSecret).toBe(true);
  });

  it("translates bearer (type only, credential left blank)", () => {
    const { patch, authNeedsSecret } = convert(`curl --oauth2-bearer abc.def https://api.example.com`);
    expect(patch.authorization).toEqual({ type: "bearer" });
    expect(authNeedsSecret).toBe(true);
  });

  it("derives bearer from Authorization header and removes the header", () => {
    const { patch } = convert(`curl https://api.example.com -H 'Authorization: Bearer tok'`);
    expect(patch.authorization).toEqual({ type: "bearer" });
    expect(patch.headers ?? []).toEqual([]);
  });

  it("derives custom-header auth from a non-Bearer/Basic Authorization header", () => {
    const { patch } = convert(`curl https://api.example.com -H 'Authorization: Token xyz'`);
    expect(patch.authorization).toEqual({ type: "custom_header", headerName: "Authorization" });
  });

  it("leaves authNeedsSecret false when there is no auth", () => {
    const { authNeedsSecret } = convert(`curl https://api.example.com`);
    expect(authNeedsSecret).toBe(false);
  });

  it("explicitly clears every curl-owned field not present in the command", () => {
    // A bare curl should still emit an `undefined` for each owned key so that
    // re-parsing wipes any prior prefill in the form.
    const { patch } = convert(`curl https://api.example.com/v1/x`);
    const ownedKeys = [
      "headers",
      "queryParams",
      "contentType",
      "json",
      "formData",
      "text",
      "xml",
      "authorization",
    ] as const;
    for (const key of ownedKeys) {
      expect(key in patch).toBe(true);
      expect(patch[key]).toBeUndefined();
    }
    expect(patch.method).toBe("GET");
    expect(patch.url).toBe("https://api.example.com/v1/x");
  });
});
