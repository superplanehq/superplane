import { describe, expect, it } from "bun:test";

import { resolveElkOutputChannels } from "./layoutGraph";

describe("resolveElkOutputChannels", () => {
  it("keeps unused metadata channels on horizontal canvases", () => {
    expect(
      resolveElkOutputChannels({
        metadataChannels: ["true", "false"],
        connectedChannels: ["true"],
        isVertical: false,
      }),
    ).toEqual(["true", "false"]);
  });

  it("drops unused metadata channels on vertical canvases", () => {
    expect(
      resolveElkOutputChannels({
        metadataChannels: ["true", "false"],
        connectedChannels: ["true"],
        isVertical: true,
      }),
    ).toEqual(["true"]);
  });

  it("keeps metadata order when both branches are connected", () => {
    expect(
      resolveElkOutputChannels({
        metadataChannels: ["true", "false"],
        connectedChannels: ["false", "true"],
        isVertical: true,
      }),
    ).toEqual(["true", "false"]);
  });

  it("falls back to default when no channels exist", () => {
    expect(
      resolveElkOutputChannels({
        metadataChannels: [],
        connectedChannels: [],
        isVertical: true,
      }),
    ).toEqual(["default"]);
  });
});
