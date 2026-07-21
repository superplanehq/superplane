import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Avatar } from "./avatar";

describe("Avatar", () => {
  it("renders the image when a source is provided", () => {
    const { container } = render(<Avatar src="https://example.com/ada.png" initials="A" alt="Ada" />);
    const img = container.querySelector("img");
    expect(img).not.toBeNull();
    expect(img!.getAttribute("src")).toBe("https://example.com/ada.png");
    // Initials are hidden behind the image until/unless it fails to load.
    expect(container.querySelector("svg text")).toBeNull();
  });

  it("falls back to initials when the image fails to load", () => {
    const { container } = render(<Avatar src="https://example.com/missing.png" initials="A" alt="Ada" />);
    const img = container.querySelector("img")!;
    fireEvent.error(img);
    expect(container.querySelector("img")).toBeNull();
    expect(container.querySelector("svg text")?.textContent).toBe("A");
  });

  it("falls back to a generic placeholder when the image fails and there are no initials", () => {
    const { container } = render(<Avatar src="https://example.com/missing.png" alt="Bot" />);
    const img = container.querySelector("img")!;
    fireEvent.error(img);
    expect(container.querySelector("img")).toBeNull();
    // Placeholder silhouette (a path, not a text badge) is rendered instead.
    expect(container.querySelector("svg path")).not.toBeNull();
    expect(container.querySelector("svg text")).toBeNull();
  });

  it("renders a generic placeholder when neither image nor initials are provided", () => {
    const { container } = render(<Avatar alt="Anon" />);
    expect(container.querySelector("img")).toBeNull();
    expect(container.querySelector("svg path")).not.toBeNull();
  });
});
