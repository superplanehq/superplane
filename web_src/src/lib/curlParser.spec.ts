import { describe, expect, it } from "vitest";
import { formatCurlCommand, parseCurlCommand } from "@/lib/curlParser";

describe("parseCurlCommand", () => {
  it("parses a simple GET request", () => {
    const result = parseCurlCommand("curl https://api.example.com/users");

    expect(result.error).toBeUndefined();
    expect(result.method).toBe("GET");
    expect(result.url).toBe("https://api.example.com/users");
    expect(result.headers).toEqual([]);
    expect(result.body).toBeUndefined();
    expect(result.queryParams).toEqual([]);
  });

  it("parses a POST request with headers and json body", () => {
    const result = parseCurlCommand(
      `curl -X POST https://api.example.com/users -H "Content-Type: application/json" -H "Authorization: Bearer token" -d '{"name":"John","meta":{"role":"admin"}}'`,
    );

    expect(result.error).toBeUndefined();
    expect(result.method).toBe("POST");
    expect(result.url).toBe("https://api.example.com/users");
    expect(result.headers).toEqual([
      { name: "Content-Type", value: "application/json" },
      { name: "Authorization", value: "Bearer token" },
    ]);
    expect(result.body).toBe('{"name":"John","meta":{"role":"admin"}}');
  });

  it("extracts query params from URL and decodes encoded values", () => {
    const result = parseCurlCommand("curl 'https://api.example.com/search?q=superplane&page=2&name=John%20Doe'");

    expect(result.error).toBeUndefined();
    expect(result.url).toBe("https://api.example.com/search");
    expect(result.queryParams).toEqual([
      { key: "q", value: "superplane" },
      { key: "page", value: "2" },
      { key: "name", value: "John Doe" },
    ]);
  });

  it("supports multiline curl commands with backslash continuations", () => {
    const result = parseCurlCommand(`curl -X POST https://api.example.com/users \
      -H "Content-Type: application/json" \
      --data-raw '{"name":"John Doe"}'`);

    expect(result.error).toBeUndefined();
    expect(result.method).toBe("POST");
    expect(result.url).toBe("https://api.example.com/users");
    expect(result.headers).toEqual([{ name: "Content-Type", value: "application/json" }]);
    expect(result.body).toBe('{"name":"John Doe"}');
  });

  it("supports inline and long-form method flags", () => {
    const inline = parseCurlCommand("curl -XPATCH https://api.example.com/users/1");
    const longForm = parseCurlCommand("curl --request=DELETE https://api.example.com/users/1");

    expect(inline.error).toBeUndefined();
    expect(inline.method).toBe("PATCH");
    expect(longForm.error).toBeUndefined();
    expect(longForm.method).toBe("DELETE");
  });

  it("supports different data flag formats", () => {
    const dataEquals = parseCurlCommand(`curl https://api.example.com/users --data='{"name":"Jane"}'`);
    const dataBinary = parseCurlCommand(
      `curl --request POST https://api.example.com/users --data-binary '{"name":"Ana"}'`,
    );

    expect(dataEquals.error).toBeUndefined();
    expect(dataEquals.body).toBe('{"name":"Jane"}');
    expect(dataEquals.method).toBe("POST");

    expect(dataBinary.error).toBeUndefined();
    expect(dataBinary.body).toBe('{"name":"Ana"}');
    expect(dataBinary.method).toBe("POST");
  });

  it("supports single-quoted header values", () => {
    const result = parseCurlCommand(
      "curl https://api.example.com/users -H 'X-Trace-Id: abc-123' -H 'X-Note: value:with:colons'",
    );

    expect(result.error).toBeUndefined();
    expect(result.headers).toEqual([
      { name: "X-Trace-Id", value: "abc-123" },
      { name: "X-Note", value: "value:with:colons" },
    ]);
  });

  it("handles quoted body edge cases", () => {
    const escapedJson = parseCurlCommand(
      `curl -X POST https://api.example.com -d '{"nested":"value\\"with\\"quotes"}'`,
    );
    const emptyBody = parseCurlCommand(`curl -X POST https://api.example.com -d ""`);

    expect(escapedJson.error).toBeUndefined();
    expect(escapedJson.body).toBe('{"nested":"value\\"with\\"quotes"}');
    expect(emptyBody.error).toBeUndefined();
    expect(emptyBody.body).toBe("");
  });

  it("handles unmatched quote segments without throwing", () => {
    const result = parseCurlCommand(`curl https://api.example.com -H "Content-Type: application/json`);

    expect(result.error).toBeUndefined();
    expect(result.url).toBe("https://api.example.com");
    expect(result.headers).toEqual([{ name: '"Content-Type', value: "application/json" }]);
  });

  it("handles URL variants including ports and fragments", () => {
    const withPort = parseCurlCommand("curl https://api.example.com:8080/users");
    const withFragment = parseCurlCommand("curl https://api.example.com/page#section");
    const urlThenMethod = parseCurlCommand("curl https://api.example.com -X GET");
    const methodThenUrl = parseCurlCommand("curl -X GET https://api.example.com");
    const localhostOnly = parseCurlCommand("curl localhost");

    expect(withPort.url).toBe("https://api.example.com:8080/users");
    expect(withFragment.url).toBe("https://api.example.com/page#section");
    expect(urlThenMethod.url).toBe("https://api.example.com");
    expect(methodThenUrl.url).toBe("https://api.example.com");
    expect(localhostOnly.url).toBe("localhost");
  });

  it("supports additional HTTP methods and normalizes casing", () => {
    expect(parseCurlCommand("curl -X PUT https://api.example.com/users/1 -d '{}'").method).toBe("PUT");
    expect(parseCurlCommand("curl -X DELETE https://api.example.com/users/1").method).toBe("DELETE");
    expect(parseCurlCommand("curl -X PATCH https://api.example.com/users/1 -d '{}'").method).toBe("PATCH");
    expect(parseCurlCommand("curl -X OPTIONS https://api.example.com").method).toBe("OPTIONS");
    expect(parseCurlCommand("curl -X HEAD https://api.example.com").method).toBe("HEAD");
    expect(parseCurlCommand("curl -X post https://api.example.com").method).toBe("POST");
  });

  it("handles header edge cases including empty and duplicate values", () => {
    const emptyValue = parseCurlCommand(`curl https://api.example.com -H "X-Empty:"`);
    const specialChars = parseCurlCommand(`curl https://api.example.com -H "X-Data: value=foo&bar=baz"`);
    const duplicateHeaders = parseCurlCommand(
      `curl https://api.example.com -H "X-Custom: value1" -H "X-Custom: value2"`,
    );
    const equalsSyntax = parseCurlCommand(`curl https://api.example.com --header="Content-Type: application/json"`);

    expect(emptyValue.headers).toEqual([{ name: "X-Empty", value: "" }]);
    expect(specialChars.headers).toEqual([{ name: "X-Data", value: "value=foo&bar=baz" }]);
    expect(duplicateHeaders.headers).toEqual([
      { name: "X-Custom", value: "value1" },
      { name: "X-Custom", value: "value2" },
    ]);
    expect(equalsSyntax.headers).toEqual([]);
  });

  it("handles body data edge cases", () => {
    const multiData = parseCurlCommand(`curl -X POST https://api.example.com -d "part1" -d "part2"`);
    const dataRawEquals = parseCurlCommand(`curl -X POST https://api.example.com --data-raw='{"test":true}'`);
    const multilineBody = parseCurlCommand(`curl -X POST https://api.example.com -d "line1\nline2"`);

    expect(multiData.body).toBe("part2");
    expect(dataRawEquals.body).toBe('{"test":true}');
    expect(multilineBody.body).toBe("line1\nline2");
  });

  it("returns an error for non-curl commands", () => {
    const result = parseCurlCommand("wget https://api.example.com/users");

    expect(result.error).toBe("Not a valid curl command");
  });

  it("returns an error when required flag values are missing", () => {
    expect(parseCurlCommand("curl -X").error).toBe("Missing method after -X flag");
    expect(parseCurlCommand("curl https://api.example.com -H").error).toBe("Missing header after -H flag");
    expect(parseCurlCommand("curl https://api.example.com -d").error).toBe("Missing body after -d flag");
  });

  it("returns an error for invalid header format", () => {
    const result = parseCurlCommand(`curl https://api.example.com/users -H "InvalidHeaderValue"`);

    expect(result.error).toBe("Invalid header format: InvalidHeaderValue");
  });

  it("handles empty and whitespace-only values safely", () => {
    expect(parseCurlCommand("").error).toBe("Not a valid curl command");
    expect(parseCurlCommand("   ").error).toBe("Not a valid curl command");
  });

  it("handles null and undefined values passed at runtime", () => {
    const nullInput = parseCurlCommand(null as unknown as string);
    const undefinedInput = parseCurlCommand(undefined as unknown as string);

    expect(nullInput.error).toBeTypeOf("string");
    expect(undefinedInput.error).toBeTypeOf("string");
  });

  it("handles malformed or partial curl commands gracefully", () => {
    const curlOnly = parseCurlCommand("curl");
    const withCommonFlags = parseCurlCommand("curl -v -s -L https://api.example.com");
    const malformedPrefix = parseCurlCommand("curlhttps://api.example.com");

    expect(curlOnly.error).toBeUndefined();
    expect(curlOnly.url).toBe("");
    expect(withCommonFlags.error).toBeUndefined();
    expect(withCommonFlags.url).toBe("https://api.example.com");
    expect(malformedPrefix.error).toBeUndefined();
    expect(malformedPrefix.url).toBe("");
  });
});

describe("formatCurlCommand", () => {
  it("formats a parsed result with method headers body and query params", () => {
    const formatted = formatCurlCommand({
      method: "POST",
      url: "https://api.example.com/users",
      headers: [
        { name: "Content-Type", value: "application/json" },
        { name: "Authorization", value: "Bearer token" },
      ],
      body: '{"name":"John"}',
      queryParams: [
        { key: "page", value: "2" },
        { key: "name", value: "John Doe" },
      ],
    });

    expect(formatted).toBe(
      'curl -X POST -H "Content-Type: application/json" -H "Authorization: Bearer token" -d "{"name":"John"}" https://api.example.com/users?page=2&name=John%20Doe',
    );
  });

  it("formats a minimal GET request", () => {
    const formatted = formatCurlCommand({
      method: "GET",
      url: "https://api.example.com/health",
      headers: [],
      queryParams: [],
    });

    expect(formatted).toBe("curl https://api.example.com/health");
  });

  it("formats special characters and quoted body values", () => {
    const formatted = formatCurlCommand({
      method: "POST",
      url: "https://api.example.com/search",
      headers: [{ name: "X-Data", value: "a=b&c=d" }],
      body: '{"key":"value with \\"quotes\\""}',
      queryParams: [{ key: "q", value: "hello world" }],
    });

    expect(formatted).toBe(
      'curl -X POST -H "X-Data: a=b&c=d" -d "{"key":"value with \\"quotes\\""}" https://api.example.com/search?q=hello%20world',
    );
  });

  it("handles empty URL and empty optional fields", () => {
    const formatted = formatCurlCommand({
      method: "GET",
      url: "",
      headers: [],
      queryParams: [],
    });

    expect(formatted).toBe("curl");
  });
});
