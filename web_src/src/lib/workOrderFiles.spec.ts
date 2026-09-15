import { afterEach, describe, expect, it } from "bun:test";

import {
  clearWorkOrderFileDownloadCache,
  isAllowedWorkOrderFile,
  isInlineWorkOrderImage,
  isReachableWorkOrderFileUrl,
  parseWorkOrderFileId,
  rewriteWorkOrderFileRefs,
  setWorkOrderFilePreviewUrl,
  resolveWorkOrderFileSrc,
  workOrderFileDownloadMap,
  workOrderFileRef,
} from "./workOrderFiles";

describe("workOrderFiles", () => {
  afterEach(() => {
    clearWorkOrderFileDownloadCache();
  });

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

  it("keeps the current URL when only its signature changes", () => {
    const id = "cccccccc-cccc-cccc-cccc-cccccccccccc";
    const first = "https://files.example/bug.png?expires=9999999999&sig=one";
    const reminted = "https://files.example/bug.png?expires=9999999999&sig=two";

    expect(workOrderFileDownloadMap([{ id, downloadUrl: first }])[id]).toBe(first);
    expect(workOrderFileDownloadMap([{ id, downloadUrl: reminted }])[id]).toBe(first);
  });

  it("replaces a SuperPlane URL that is about to expire", () => {
    const id = "dddddddd-dddd-dddd-dddd-dddddddddddd";
    const expiring = `https://files.example/bug.png?expires=${Math.floor(Date.now() / 1000) + 10}&sig=old`;
    const next = `https://files.example/bug.png?expires=${Math.floor(Date.now() / 1000) + 3600}&sig=new`;

    expect(workOrderFileDownloadMap([{ id, downloadUrl: expiring }])[id]).toBe(expiring);
    expect(workOrderFileDownloadMap([{ id, downloadUrl: next }])[id]).toBe(next);
  });

  it("replaces a GCS URL that is about to expire", () => {
    const id = "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee";
    const signedAt = new Date().toISOString().replace(/[-:]|\.\d{3}/g, "");
    const expiring = `https://storage.googleapis.com/tasks/bug.png?X-Goog-Date=${signedAt}&X-Goog-Expires=10&X-Goog-Signature=old`;
    const next = `https://storage.googleapis.com/tasks/bug.png?X-Goog-Date=${signedAt}&X-Goog-Expires=3600&X-Goog-Signature=new`;

    expect(workOrderFileDownloadMap([{ id, downloadUrl: expiring }])[id]).toBe(expiring);
    expect(workOrderFileDownloadMap([{ id, downloadUrl: next }])[id]).toBe(next);
  });

  it("replaces a URL when the resource path changes", () => {
    const id = "ffffffff-ffff-ffff-ffff-ffffffffffff";
    const first = "https://files.example/old.png?expires=9999999999&sig=one";
    const next = "https://files.example/new.png?expires=9999999999&sig=two";

    expect(workOrderFileDownloadMap([{ id, downloadUrl: first }])[id]).toBe(first);
    expect(workOrderFileDownloadMap([{ id, downloadUrl: next }])[id]).toBe(next);
  });

  it("evicts the oldest URL after caching 200 files", () => {
    const firstId = "file-0";
    const first = "https://files.example/file-0.png?expires=9999999999&sig=one";
    const reminted = "https://files.example/file-0.png?expires=9999999999&sig=two";
    workOrderFileDownloadMap([{ id: firstId, downloadUrl: first }]);

    for (let index = 1; index <= 200; index += 1) {
      workOrderFileDownloadMap([
        { id: `file-${index}`, downloadUrl: `https://files.example/file-${index}.png?expires=9999999999` },
      ]);
    }

    expect(workOrderFileDownloadMap([{ id: firstId, downloadUrl: reminted }])[firstId]).toBe(reminted);
  });
});
