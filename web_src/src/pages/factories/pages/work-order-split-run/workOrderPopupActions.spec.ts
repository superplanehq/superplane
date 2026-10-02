import { describe, it, expect } from "bun:test";

import { stripFileReferences } from "./workOrderPopupActions";

describe("stripFileReferences", () => {
  it("removes markdown image references with sp-file:// URLs", () => {
    const description =
      "Here is an image ![screenshot](sp-file://550e8400-e29b-41d4-a716-446655440000) in the text";
    const result = stripFileReferences(description);
    expect(result).toBe("Here is an image  in the text");
  });

  it("removes markdown link references with sp-file:// URLs", () => {
    const description =
      "Here is a [link](sp-file://550e8400-e29b-41d4-a716-446655440000) in the text";
    const result = stripFileReferences(description);
    expect(result).toBe("Here is a  in the text");
  });

  it("preserves normal HTTP links", () => {
    const description = "Here is a [link](https://example.com) in the text";
    const result = stripFileReferences(description);
    expect(result).toBe("Here is a [link](https://example.com) in the text");
  });

  it("preserves normal HTTPS image links", () => {
    const description = "Here is an image ![alt](https://example.com/image.png) in the text";
    const result = stripFileReferences(description);
    expect(result).toBe("Here is an image ![alt](https://example.com/image.png) in the text");
  });

  it("removes HTML img src with sp-file:// URLs", () => {
    const description =
      'Here is an image <img src="sp-file://550e8400-e29b-41d4-a716-446655440000" /> in the text';
    const result = stripFileReferences(description);
    expect(result).toBe('Here is an image <img src="" /> in the text');
  });

  it("removes HTML href with sp-file:// URLs", () => {
    const description =
      'Here is a <a href="sp-file://550e8400-e29b-41d4-a716-446655440000">link</a> in the text';
    const result = stripFileReferences(description);
    expect(result).toBe('Here is a <a href="">link</a> in the text');
  });

  it("preserves normal HTTP href URLs", () => {
    const description =
      'Here is a <a href="https://example.com">link</a> in the text';
    const result = stripFileReferences(description);
    expect(result).toBe(
      'Here is a <a href="https://example.com">link</a> in the text',
    );
  });

  it("removes empty markdown syntax left after stripping sp-file:// references", () => {
    const description = "Before ![]() after";
    const result = stripFileReferences(description);
    expect(result).toBe("Before  after");
  });

  it("handles multiple sp-file:// references in one description", () => {
    const description =
      "Image 1: ![](sp-file://550e8400-e29b-41d4-a716-446655440000) and image 2: ![](sp-file://660e8400-e29b-41d4-a716-446655440001)";
    const result = stripFileReferences(description);
    expect(result).toBe("Image 1:  and image 2: ");
  });

  it("preserves surrounding text and normal markdown", () => {
    const description =
      "# Title\n\nSome **bold** text with ![pasted](sp-file://550e8400-e29b-41d4-a716-446655440000) and [normal link](https://example.com).";
    const result = stripFileReferences(description);
    expect(result).toBe(
      "# Title\n\nSome **bold** text with  and [normal link](https://example.com).",
    );
  });

  it("handles sp-file:// URLs with single quotes in HTML", () => {
    const description =
      "Here is an image <img src='sp-file://550e8400-e29b-41d4-a716-446655440000' /> in the text";
    const result = stripFileReferences(description);
    expect(result).toBe("Here is an image <img src='' /> in the text");
  });

  it("handles mixed case HTML attributes", () => {
    const description =
      'Here is an image <IMG SRC="sp-file://550e8400-e29b-41d4-a716-446655440000" /> in the text';
    const result = stripFileReferences(description);
    expect(result).toBe('Here is an image <IMG SRC="" /> in the text');
  });

  it("preserves empty strings", () => {
    const description = "";
    const result = stripFileReferences(description);
    expect(result).toBe("");
  });
});
