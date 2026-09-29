import { describe, expect, it } from "bun:test";
import {
  isBenignLiveLogWait,
  LIVE_LOG_ERROR_CODE_HEADER,
  LIVE_LOG_SESSION_NOT_READY_CODE,
  LiveLogRequestError,
  liveLogRequestErrorFromResponse,
} from "./liveLogErrors";

describe("live log wait detection", () => {
  it("treats a not-ready session by error code, not copy", () => {
    const error = liveLogRequestErrorFromResponse(
      new Response("Wording can change without changing the wait.", {
        status: 404,
        statusText: "Not Found",
        headers: { [LIVE_LOG_ERROR_CODE_HEADER]: LIVE_LOG_SESSION_NOT_READY_CODE },
      }),
      "Wording can change without changing the wait.",
    );

    expect(error.code).toBe(LIVE_LOG_SESSION_NOT_READY_CODE);
    expect(isBenignLiveLogWait(error)).toBe(true);
  });

  it("does not treat a 404 as a wait when the error code is missing", () => {
    const error = liveLogRequestErrorFromResponse(
      new Response("Logs are not available for this execution yet. Check again shortly.", {
        status: 404,
        statusText: "Not Found",
      }),
      "Logs are not available for this execution yet. Check again shortly.",
    );

    expect(error.code).toBeUndefined();
    expect(isBenignLiveLogWait(error)).toBe(false);
  });

  it("treats a CloudWatch log-stream-not-found broker error as a wait", () => {
    expect(
      isBenignLiveLogWait(
        new Error(
          "operation error CloudWatch Logs: GetLogEvents, ResourceNotFoundException: The specified log stream does not exist.",
        ),
      ),
    ).toBe(true);
  });

  it("does not treat a CloudWatch log-group-not-found broker error as a wait", () => {
    expect(
      isBenignLiveLogWait(
        new Error(
          "operation error CloudWatch Logs: GetLogEvents, ResourceNotFoundException: The specified log group does not exist.",
        ),
      ),
    ).toBe(false);
  });

  it("does not treat an ordinary request failure as a wait", () => {
    expect(isBenignLiveLogWait(new LiveLogRequestError("Failed to fetch"))).toBe(false);
  });
});
