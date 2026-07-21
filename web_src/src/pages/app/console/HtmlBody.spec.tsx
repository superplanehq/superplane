import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { HtmlBody } from "./HtmlBody";

describe("HtmlBody broken avatar fallback", () => {
  it("replaces a failed avatar image with an initials fallback derived from alt", () => {
    const { container } = render(
      <HtmlBody
        body={'<img class="avatar avatar-image" src="https://github.com/some-bot[bot].png" alt="Some Bot" />'}
        vars={{}}
      />,
    );
    const img = container.querySelector("img.avatar-image") as HTMLImageElement;
    expect(img).not.toBeNull();

    fireEvent.error(img);

    expect(container.querySelector("img")).toBeNull();
    const fallback = container.querySelector("div.avatar-fallback");
    expect(fallback).not.toBeNull();
    expect(fallback!.classList.contains("avatar")).toBe(true);
    expect(fallback!.textContent).toBe("S");
  });

  it("uses a silhouette (aria-hidden) fallback when there is no alt text", () => {
    const { container } = render(
      <HtmlBody body={'<img class="avatar avatar-image" src="https://github.com/x.png" alt="" />'} vars={{}} />,
    );
    const img = container.querySelector("img.avatar-image") as HTMLImageElement;
    fireEvent.error(img);

    const fallback = container.querySelector("div.avatar-fallback");
    expect(fallback).not.toBeNull();
    expect(fallback!.textContent).toBe("");
    expect(fallback!.getAttribute("aria-hidden")).toBe("true");
  });

  it("leaves non-avatar images untouched when they fail", () => {
    const { container } = render(<HtmlBody body={'<img src="https://example.com/logo.png" alt="Logo" />'} vars={{}} />);
    const img = container.querySelector("img") as HTMLImageElement;
    fireEvent.error(img);
    // Author-authored images are not avatar images, so we do not swap them.
    expect(container.querySelector("img")).not.toBeNull();
    expect(container.querySelector("div.avatar-fallback")).toBeNull();
  });
});
