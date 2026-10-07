import { describe, expect, it } from "bun:test";

import { hostedVideoFieldError, hostedVideoFromClipboard, parseHostedVideoUrl } from "./hostedVideo";

describe("parseHostedVideoUrl", () => {
  it("builds provider embeds from the video id", () => {
    expect(parseHostedVideoUrl("https://www.youtube.com/watch?v=dQw4w9WgXcQ")).toMatchObject({
      providerName: "YouTube",
      embedUrl: "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ",
    });
    expect(parseHostedVideoUrl("https://youtu.be/dQw4w9WgXcQ")?.embedUrl).toBe(
      "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ",
    );
    expect(parseHostedVideoUrl("https://www.youtube.com/shorts/dQw4w9WgXcQ")?.embedUrl).toBe(
      "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ",
    );
    expect(parseHostedVideoUrl("https://vimeo.com/123456789")?.embedUrl).toBe(
      "https://player.vimeo.com/video/123456789",
    );
    expect(parseHostedVideoUrl("https://www.loom.com/share/0123456789abcdef0123456789abcdef")?.embedUrl).toBe(
      "https://www.loom.com/embed/0123456789abcdef0123456789abcdef",
    );
  });

  it("uses a card for CleanShot and rejects unknown or insecure links", () => {
    const cleanshot = parseHostedVideoUrl("https://cln.sh/abcd1234");
    expect(cleanshot).toMatchObject({ providerName: "CleanShot", embedUrl: null, pageUrl: "https://cln.sh/abcd1234" });
    expect(parseHostedVideoUrl("https://example.com/watch?v=dQw4w9WgXcQ")).toBeNull();
    expect(parseHostedVideoUrl("http://www.youtube.com/watch?v=dQw4w9WgXcQ")).toBeNull();
    expect(parseHostedVideoUrl("https://www.youtube.com/playlist?list=PL123")).toBeNull();
    expect(hostedVideoFromClipboard("see https://youtu.be/dQw4w9WgXcQ")).toBeNull();
    expect(hostedVideoFromClipboard("  https://youtu.be/dQw4w9WgXcQ\n")?.providerId).toBe("youtube");
  });

  it("rejects an unsupported host as soon as the value is a URL", () => {
    expect(hostedVideoFieldError("https://example.com/video")).toBe(
      "Use an HTTPS YouTube, Vimeo, Loom, or CleanShot link.",
    );
    expect(hostedVideoFieldError("https://www.youtube.com/watch?v=dQw4w9WgXcQ")).toBe("");
    expect(hostedVideoFieldError("you")).toBe("");
  });
});
