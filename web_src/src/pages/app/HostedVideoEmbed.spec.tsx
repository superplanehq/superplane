import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { parseHostedVideoUrl } from "@/lib/hostedVideo";

import { HostedVideoEmbed } from "./HostedVideoEmbed";

describe("HostedVideoEmbed", () => {
  it("renders a provider embed from the video id", () => {
    const video = parseHostedVideoUrl("https://www.youtube.com/watch?v=dQw4w9WgXcQ");
    expect(video).not.toBeNull();
    render(<HostedVideoEmbed video={video!} />);

    const frame = document.querySelector("iframe");
    expect(frame).toHaveAttribute("src", "https://www.youtube-nocookie.com/embed/dQw4w9WgXcQ");
    expect(frame).not.toHaveAttribute("src", "https://www.youtube.com/watch?v=dQw4w9WgXcQ");
  });

  it("renders a CleanShot card with an open link", () => {
    const video = parseHostedVideoUrl("https://cln.sh/abcd1234");
    expect(video).not.toBeNull();
    render(<HostedVideoEmbed video={video!} />);

    expect(document.querySelector("iframe")).toBeNull();
    expect(screen.getByText("CleanShot")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open video" })).toHaveAttribute("href", "https://cln.sh/abcd1234");
  });
});
