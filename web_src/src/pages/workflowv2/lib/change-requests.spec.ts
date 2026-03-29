import { describe, expect, it } from "vitest";
import type { CanvasesCanvasChangeRequest, CanvasesCanvasVersion } from "@/api-client";
import { buildChangeRequestVersionRowsForStatus } from "./change-requests";

describe("buildChangeRequestVersionRowsForStatus", () => {
  it("filters by status and resolves missing embedded versions from the visible versions list", () => {
    const visibleVersion: CanvasesCanvasVersion = {
      metadata: {
        id: "version-1",
      },
      spec: {
        nodes: [],
      },
    };

    const changeRequests: CanvasesCanvasChangeRequest[] = [
      {
        metadata: {
          id: "request-1",
          status: "STATUS_OPEN",
          versionId: "version-1",
          updatedAt: "2026-03-29T11:00:00.000Z",
        },
      },
      {
        metadata: {
          id: "request-2",
          status: "STATUS_REJECTED",
          versionId: "version-2",
          updatedAt: "2026-03-29T10:00:00.000Z",
        },
      },
    ];

    const rows = buildChangeRequestVersionRowsForStatus(changeRequests, [visibleVersion], "open");

    expect(rows).toHaveLength(1);
    expect(rows[0]).toMatchObject({
      changeRequest: {
        metadata: {
          id: "request-1",
        },
      },
      version: visibleVersion,
    });
  });

  it("keeps only the newest matching request for a version", () => {
    const changeRequests: CanvasesCanvasChangeRequest[] = [
      {
        metadata: {
          id: "request-old",
          status: "STATUS_OPEN",
          updatedAt: "2026-03-29T09:00:00.000Z",
        },
        version: {
          metadata: {
            id: "version-1",
          },
          spec: {
            nodes: [],
          },
        },
      },
      {
        metadata: {
          id: "request-new",
          status: "STATUS_OPEN",
          updatedAt: "2026-03-29T12:00:00.000Z",
        },
        version: {
          metadata: {
            id: "version-1",
          },
          spec: {
            nodes: [],
          },
        },
      },
    ];

    const rows = buildChangeRequestVersionRowsForStatus(changeRequests, [], "open");

    expect(rows).toHaveLength(1);
    expect(rows[0].changeRequest.metadata?.id).toBe("request-new");
  });

  it("skips requests whose version cannot be resolved", () => {
    const changeRequests: CanvasesCanvasChangeRequest[] = [
      {
        metadata: {
          id: "request-1",
          status: "STATUS_OPEN",
          versionId: "missing-version",
          updatedAt: "2026-03-29T11:00:00.000Z",
        },
      },
    ];

    const rows = buildChangeRequestVersionRowsForStatus(changeRequests, [], "open");

    expect(rows).toEqual([]);
  });
});
