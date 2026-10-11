import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

import { HostedVideoLinkField } from "./HostedVideoLinkField";

describe("HostedVideoLinkField", () => {
  it("adds a video link without submitting the parent form", () => {
    const onAdd = vi.fn(() => true);
    const onSubmit = vi.fn();
    render(
      <form onSubmit={onSubmit}>
        <HostedVideoLinkField onAdd={onAdd} />
        <button type="submit">Create task</button>
      </form>,
    );

    fireEvent.click(screen.getByTestId("hosted-video-link-toggle"));
    const input = screen.getByTestId("hosted-video-link-input");
    const label = screen.getByTestId("hosted-video-link-label");
    expect(label).toHaveTextContent("Video link");
    expect(label.tagName).toBe("LABEL");
    expect(label).toHaveAttribute("for", input.id);
    fireEvent.change(input, {
      target: { value: "https://www.youtube.com/watch?v=dQw4w9WgXcQ" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Add" }));

    expect(onAdd).toHaveBeenCalledOnce();
    expect(onSubmit).not.toHaveBeenCalled();
  });

  it("does not submit the parent form when Enter is pressed in the link field", () => {
    const onAdd = vi.fn(() => true);
    const onSubmit = vi.fn();
    render(
      <form onSubmit={onSubmit}>
        <HostedVideoLinkField onAdd={onAdd} />
      </form>,
    );

    fireEvent.click(screen.getByTestId("hosted-video-link-toggle"));
    fireEvent.change(screen.getByTestId("hosted-video-link-input"), {
      target: { value: "https://www.youtube.com/watch?v=dQw4w9WgXcQ" },
    });
    fireEvent.keyDown(screen.getByTestId("hosted-video-link-input"), { key: "Enter", code: "Enter" });

    expect(onAdd).toHaveBeenCalledOnce();
    expect(onSubmit).not.toHaveBeenCalled();
  });
});
