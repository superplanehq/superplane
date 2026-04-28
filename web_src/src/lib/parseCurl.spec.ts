import { describe, expect, it } from "vitest";
import { parseCurl } from "./parseCurl";

describe("parseCurl", () => {
  it("parses simple GET request", () => {
    const result = parseCurl("curl https://api.example.com/users");
    expect(result.success).toBe(true);
    expect(result.request).toMatchObject({
      method: "GET",
      url: "https://api.example.com/users",
    });
  });

  it("parses POST with JSON body", () => {
    const result = parseCurl(
      `curl -X POST https://api.example.com/users -H "Content-Type: application/json" -d '{"name":"John"}'`,
    );

    expect(result.success).toBe(true);
    expect(result.request).toMatchObject({
      method: "POST",
      url: "https://api.example.com/users",
      contentType: "application/json",
      json: { name: "John" },
    });
  });

  it("defaults method to POST when body is present", () => {
    const result = parseCurl(`curl https://api.example.com/users -d '{"name":"John"}'`);
    expect(result.success).toBe(true);
    expect(result.request?.method).toBe("POST");
  });

  it("extracts headers and supports colon in value", () => {
    const result = parseCurl(
      `curl https://api.example.com -H "Authorization: Bearer abc:def" -H "X-Trace: 123"`,
    );
    expect(result.success).toBe(true);
    expect(result.request?.headers).toEqual([
      { name: "Authorization", value: "Bearer abc:def" },
      { name: "X-Trace", value: "123" },
    ]);
  });

  it("extracts query params from URL including empty values", () => {
    const result = parseCurl("curl 'https://api.example.com/search?q=test&empty='");
    expect(result.success).toBe(true);
    expect(result.request?.queryParams).toEqual([
      { key: "q", value: "test" },
      { key: "empty", value: "" },
    ]);
  });

  it("parses data-urlencode as form data", () => {
    const result = parseCurl("curl https://api.example.com --data-urlencode user=john --data-urlencode active=true");
    expect(result.success).toBe(true);
    expect(result.request).toMatchObject({
      method: "POST",
      contentType: "application/x-www-form-urlencoded",
      formData: [
        { key: "user", value: "john" },
        { key: "active", value: "true" },
      ],
    });
  });

  it("handles multiline curl with line continuation", () => {
    const result = parseCurl("curl https://api.example.com \\\n-H 'Accept: application/json'");
    expect(result.success).toBe(true);
    expect(result.request?.headers).toEqual([{ name: "Accept", value: "application/json" }]);
  });

  it("returns error for null input", () => {
    const result = parseCurl(null);
    expect(result.success).toBe(false);
    expect(result.errors).toContain("Input is required");
  });

  it("returns error for undefined input", () => {
    const result = parseCurl(undefined);
    expect(result.success).toBe(false);
    expect(result.errors).toContain("Input is required");
  });

  it("returns error for empty string input", () => {
    const result = parseCurl("");
    expect(result.success).toBe(false);
    expect(result.errors).toContain("Input is required");
  });

  it("returns error for whitespace-only input", () => {
    const result = parseCurl("   ");
    expect(result.success).toBe(false);
    expect(result.errors).toContain("Input is required");
  });

  it("returns invalid curl command when only curl keyword is provided", () => {
    const result = parseCurl("curl");
    expect(result.success).toBe(false);
    expect(result.errors).toContain("Invalid curl command");
  });

  it("keeps empty header value", () => {
    const result = parseCurl(`curl https://api.example.com -H "X-Empty:"`);
    expect(result.success).toBe(true);
    expect(result.request?.headers).toEqual([{ name: "X-Empty", value: "" }]);
  });

  it("handles empty body", () => {
    const result = parseCurl(`curl https://api.example.com -d ""`);
    expect(result.success).toBe(true);
    expect(result.request).toMatchObject({
      method: "POST",
      text: "",
    });
  });

  it("handles JSON body with null values", () => {
    const result = parseCurl(`curl https://api.example.com -d '{"name":null,"age":1}'`);
    expect(result.success).toBe(true);
    expect(result.request?.json).toEqual({ name: null, age: 1 });
  });

  it("handles malformed JSON body gracefully", () => {
    const result = parseCurl(`curl https://api.example.com -d '{"name":}'`);
    expect(result.success).toBe(true);
    expect(result.warnings).toContain("Malformed JSON body parsed as plain text");
    expect(result.request?.text).toBe('{"name":}');
  });

  it("adds warnings for unsupported flags", () => {
    const result = parseCurl("curl https://api.example.com --cert cert.pem --key key.pem");
    expect(result.success).toBe(true);
    expect(result.warnings).toEqual(
      expect.arrayContaining(["Unsupported flag ignored: --cert", "Unsupported flag ignored: --key"]),
    );
  });
});
