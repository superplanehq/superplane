import { describe, expect, it } from "bun:test";

import {
  isAllowedWorkOrderFile,
  isInlineWorkOrderImage,
  isReachableWorkOrderFileUrl,
  parseWorkOrderFileId,
  rewriteWorkOrderFileRefs,
  setWorkOrderFilePreviewUrl,
  resolveWorkOrderFileSrc,
  workOrderFileRef,
} from "./workOrderFiles";

describe("workOrderFiles", () => {
  it("parses and rewrites stored file refs", () => {
    const id = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa";
    expect(parseWorkOrderFileId(workOrderFileRef(id))).toBe(id);

    const markdown = `![bug](${workOrderFileRef(id)})`;
    expect(rewriteWorkOrderFileRefs(markdown, [{ id, downloadUrl: "https://cdn.example/bug.png" }])).toBe(
      "![bug](https://cdn.example/bug.png)",
    );
  });

  it("resolves editor previews before minted download URLs", () => {
    const id = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb";
    setWorkOrderFilePreviewUrl(id, "blob:preview");
    expect(resolveWorkOrderFileSrc(workOrderFileRef(id), { [id]: "https://cdn.example/x.png" })).toBe("blob:preview");
  });

  it("accepts allowed images and rejects other types", () => {
    expect(isAllowedWorkOrderFile(new File(["x"], "a.png", { type: "image/png" }))).toBe(true);
    expect(isAllowedWorkOrderFile(new File(["x"], "a.mp4", { type: "video/mp4" }))).toBe(false);
    expect(isInlineWorkOrderImage("image/jpeg")).toBe(true);
    expect(isInlineWorkOrderImage("application/pdf")).toBe(false);
  });

  it("treats http, https, and blob URLs as reachable image sources", () => {
    expect(isReachableWorkOrderFileUrl("https://cdn.example/bug.png")).toBe(true);
    expect(isReachableWorkOrderFileUrl("http://files.test/bug.png")).toBe(true);
    expect(isReachableWorkOrderFileUrl("blob:preview")).toBe(true);
    expect(isReachableWorkOrderFileUrl("sp-file://aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")).toBe(false);
    expect(isReachableWorkOrderFileUrl("")).toBe(false);
  });
});
