import { afterEach, describe, expect, it } from "bun:test";

import {
  clearWorkOrderFileDownloadCache,
  isAllowedWorkOrderFile,
  isInlineWorkOrderImage,
  isReachableWorkOrderFileUrl,
  parseWorkOrderFileId,
  resolveWorkOrderFileMimeType,
  rewriteWorkOrderFileRefs,
  setWorkOrderFilePreviewUrl,
  resolveWorkOrderFileSrc,
  workOrderFileDownloadMap,
  workOrderFileDownloadUrlIsFresh,
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

  it("accepts the allowed text data files", () => {
    expect(isAllowedWorkOrderFile(new File(["x"], "data.json", { type: "application/json" }))).toBe(true);
    expect(isAllowedWorkOrderFile(new File(["x"], "rows.csv", { type: "text/csv" }))).toBe(true);
    expect(isAllowedWorkOrderFile(new File(["x"], "config.yaml", { type: "application/yaml" }))).toBe(true);
    expect(isAllowedWorkOrderFile(new File(["x"], "config.yaml", { type: "text/yaml" }))).toBe(true);
    expect(isAllowedWorkOrderFile(new File(["x"], "config.yml", { type: "application/x-yaml" }))).toBe(true);
    expect(isAllowedWorkOrderFile(new File(["x"], "clip.mp4", { type: "video/mp4" }))).toBe(false);
    expect(isAllowedWorkOrderFile(new File(["x"], "page.html", { type: "text/html" }))).toBe(false);
  });

  it("accepts Word and HEIC files without treating them as inline images", () => {
    const files = [
      ["document.doc", "application/msword"],
      ["document.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"],
      ["photo.heic", "image/heic"],
      ["photo.heif", "image/heif"],
    ] as const;

    for (const [name, type] of files) {
      expect(isAllowedWorkOrderFile(new File(["x"], name, { type }))).toBe(true);
      expect(isAllowedWorkOrderFile(new File(["x"], name, { type: "" }))).toBe(true);
      expect(isInlineWorkOrderImage(type)).toBe(false);
    }
  });

  it("normalizes CSV browser aliases without accepting Excel files", () => {
    const aliases = ["text/x-csv", "application/csv", "text/comma-separated-values"];
    for (const type of aliases) {
      expect(resolveWorkOrderFileMimeType({ name: "rows.csv", type })).toBe("text/csv");
    }

    expect(resolveWorkOrderFileMimeType({ name: "rows.csv", type: "application/vnd.ms-excel" })).toBe("text/csv");
    expect(isAllowedWorkOrderFile(new File(["x"], "rows.csv", { type: "application/vnd.ms-excel" }))).toBe(true);
    expect(isAllowedWorkOrderFile(new File(["x"], "sheet.xls", { type: "application/vnd.ms-excel" }))).toBe(false);
  });

  it("resolves the MIME type from the filename when the browser sends none", () => {
    expect(resolveWorkOrderFileMimeType({ name: "notes.txt", type: "" })).toBe("text/plain");
    expect(resolveWorkOrderFileMimeType({ name: "readme.md", type: "application/octet-stream" })).toBe("text/markdown");
    expect(resolveWorkOrderFileMimeType({ name: "data.json", type: "" })).toBe("application/json");
    expect(resolveWorkOrderFileMimeType({ name: "rows.csv", type: "" })).toBe("text/csv");
    expect(resolveWorkOrderFileMimeType({ name: "config.yaml", type: "" })).toBe("application/yaml");
    expect(resolveWorkOrderFileMimeType({ name: "config.yml", type: "" })).toBe("application/yaml");
    expect(resolveWorkOrderFileMimeType({ name: "report.pdf", type: "" })).toBe("application/pdf");
    expect(resolveWorkOrderFileMimeType({ name: "photo.jpg", type: "" })).toBe("image/jpeg");
    expect(resolveWorkOrderFileMimeType({ name: "shot.png", type: "image/png" })).toBe("image/png");
    expect(resolveWorkOrderFileMimeType({ name: "archive", type: "" })).toBe("");
    expect(isAllowedWorkOrderFile(new File(["x"], "notes.md", { type: "" }))).toBe(true);
    expect(isAllowedWorkOrderFile(new File(["x"], "archive.zip", { type: "" }))).toBe(false);
    expect(isAllowedWorkOrderFile(new File(["x"], "drawing.svg", { type: "image/svg+xml" }))).toBe(false);
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

  it("treats unsigned URLs and far-future signed URLs as fresh", () => {
    expect(workOrderFileDownloadUrlIsFresh("https://files.example/bug.png")).toBe(true);
    expect(workOrderFileDownloadUrlIsFresh("https://files.example/bug.png?expires=9999999999&sig=one")).toBe(true);
  });

  it("treats a SuperPlane URL that is about to expire as stale", () => {
    const expiring = `https://files.example/bug.png?expires=${Math.floor(Date.now() / 1000) + 10}&sig=old`;
    expect(workOrderFileDownloadUrlIsFresh(expiring)).toBe(false);
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
