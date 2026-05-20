import { describe, expect, it } from "vitest";
import { parseCurl } from "@/lib/parseCurl";

describe("parseCurl", () => {
  it("parses a bare GET with just a URL", () => {
    const r = parseCurl("curl https://api.example.com/v1/things");
    expect(r.method).toBe("GET");
    expect(r.url).toBe("https://api.example.com/v1/things");
    expect(r.headers).toEqual([]);
    expect(r.queryParams).toEqual([]);
    expect(r.body).toBeUndefined();
    expect(r.authorization).toBeUndefined();
  });

  it("picks up -X and -H", () => {
    const r = parseCurl(`curl -X POST 'https://api.example.com/orders' -H 'X-Trace-Id: abc-123'`);
    expect(r.method).toBe("POST");
    expect(r.url).toBe("https://api.example.com/orders");
    expect(r.headers).toEqual([{ name: "X-Trace-Id", value: "abc-123" }]);
  });

  it("handles headers with spaces in the value", () => {
    const r = parseCurl(
      `curl https://api.example.com -H 'User-Agent: curl 8.5 (linux)' -H "Accept-Language: en-US, fr-CA"`,
    );
    expect(r.headers).toEqual([
      { name: "User-Agent", value: "curl 8.5 (linux)" },
      { name: "Accept-Language", value: "en-US, fr-CA" },
    ]);
  });

  it("supports double-quoted values with escapes", () => {
    const r = parseCurl(
      `curl -X POST https://api.example.com -H "Content-Type: application/json" -d "{\\"sku\\":\\"A1\\"}"`,
    );
    expect(r.method).toBe("POST");
    expect(r.body).toBe(`{"sku":"A1"}`);
    expect(r.headers).toEqual([{ name: "Content-Type", value: "application/json" }]);
  });

  it("collects every --data-urlencode pair into the body", () => {
    const r = parseCurl(
      `curl -X POST https://api.example.com/login --data-urlencode 'user=jane' --data-urlencode 'pass=hunter2'`,
    );
    expect(r.bodyFlag).toBe("--data-urlencode");
    expect(r.body).toBe("user=jane&pass=hunter2");
  });

  it("matches --json body and records the flag", () => {
    const r = parseCurl(`curl --json '{"foo":1}' https://api.example.com`);
    expect(r.bodyFlag).toBe("--json");
    expect(r.body).toBe('{"foo":1}');
    // body present → method falls back to POST
    expect(r.method).toBe("POST");
  });

  it("extracts basic auth from -u", () => {
    const r = parseCurl(`curl -u alice:s3cret https://api.example.com`);
    expect(r.authorization).toEqual({
      type: "basic_auth",
      username: "alice",
      password: "s3cret",
    });
  });

  it("extracts bearer from --oauth2-bearer", () => {
    const r = parseCurl(`curl --oauth2-bearer abc.def.ghi https://api.example.com`);
    expect(r.authorization).toEqual({ type: "bearer", token: "abc.def.ghi" });
  });

  it("falls back to Authorization header when no flag is present", () => {
    const r = parseCurl(`curl https://api.example.com -H 'Authorization: Bearer sk_live_123'`);
    expect(r.authorization).toEqual({ type: "bearer", token: "sk_live_123" });
    // header is consumed
    expect(r.headers.find((h) => h.name === "Authorization")).toBeUndefined();
  });

  it("decodes Basic auth header into username/password", () => {
    const b64 = btoa("alice:s3cret");
    const r = parseCurl(`curl https://api.example.com -H 'Authorization: Basic ${b64}'`);
    expect(r.authorization).toEqual({
      type: "basic_auth",
      username: "alice",
      password: "s3cret",
    });
  });

  it("treats non-Bearer / non-Basic Authorization headers as custom", () => {
    const r = parseCurl(`curl https://api.example.com -H 'Authorization: Token xyz'`);
    expect(r.authorization).toEqual({
      type: "custom_header",
      headerName: "Authorization",
      value: "Token xyz",
    });
  });

  it("splits inline query string into queryParams and strips it from url", () => {
    const r = parseCurl(`curl 'https://api.example.com/search?q=shoes&limit=10'`);
    expect(r.url).toBe("https://api.example.com/search");
    expect(r.queryParams).toEqual([
      { key: "q", value: "shoes" },
      { key: "limit", value: "10" },
    ]);
  });

  it("appends --url-query flags to queryParams", () => {
    const r = parseCurl(`curl https://api.example.com --url-query 'q=red shoes'`);
    expect(r.queryParams).toEqual([{ key: "q", value: "red shoes" }]);
  });

  it("handles \\-newline line continuations", () => {
    const r = parseCurl(
      `curl -X POST 'https://api.example.com/orders' \\\n  -H 'X-Trace-Id: abc' \\\n  --json '{"x":1}'`,
    );
    expect(r.method).toBe("POST");
    expect(r.headers).toEqual([{ name: "X-Trace-Id", value: "abc" }]);
    expect(r.body).toBe('{"x":1}');
  });

  it("does not over-match across multiple -d flags (regression)", () => {
    const r = parseCurl(`curl -X POST https://api.example.com -d 'a=1' -d 'b=2'`);
    // first non-urlencode -d wins; no greedy swallow into a single blob
    expect(r.body).toBe("a=1");
  });

  it("accepts a bare hostname (no scheme)", () => {
    const r = parseCurl(`curl example.com`);
    expect(r.url).toBe("example.com");
  });

  it("accepts host:port and IP-style URLs", () => {
    expect(parseCurl(`curl localhost:8080/api`).url).toBe("localhost:8080/api");
    expect(parseCurl(`curl 192.168.1.1/api`).url).toBe("192.168.1.1/api");
  });

  it("skips boolean options before the URL", () => {
    // -L (follow redirects) and -k (insecure) take no value, so the next
    // positional argument is the URL.
    const r = parseCurl(`curl -L -k example.com`);
    expect(r.url).toBe("example.com");
  });

  it("skips value-taking options and finds the positional URL", () => {
    const r = parseCurl(`curl -X POST -H 'X-Foo: bar' example.com`);
    expect(r.method).toBe("POST");
    expect(r.url).toBe("example.com");
  });

  it("supports --opt=value form when locating the URL", () => {
    const r = parseCurl(`curl --max-time=10 example.com`);
    expect(r.url).toBe("example.com");
  });

  it("supports a scheme-less URL with an inline query string", () => {
    const r = parseCurl(`curl 'example.com?q=1&limit=10'`);
    expect(r.url).toBe("example.com");
    expect(r.queryParams).toEqual([
      { key: "q", value: "1" },
      { key: "limit", value: "10" },
    ]);
  });
});
