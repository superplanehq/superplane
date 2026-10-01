import { fireEvent, render } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { GitHubAttachmentMedia } from "./GitHubAttachmentMedia";

describe("GitHubAttachmentMedia", () => {
  it("plays a new video after an earlier attachment was an image", () => {
    const image = "https://github.com/user-attachments/assets/image";
    const video = "https://github.com/user-attachments/assets/video";
    const { rerender } = render(<GitHubAttachmentMedia src={image} />);

    fireEvent.error(document.querySelector("video")!);

    expect(document.querySelector("video")).toBeNull();
    expect(document.querySelector("img")).toHaveAttribute("src", image);

    rerender(<GitHubAttachmentMedia src={video} />);

    expect(document.querySelector("img")).toBeNull();
    expect(document.querySelector("video")).toHaveAttribute("src", video);
  });
});
