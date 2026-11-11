import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, describe, expect, it, vi } from "bun:test";
import { client } from "@/api-client/client.gen";
import { renderSettings, waitForPopupDismissal } from "./ReviewRequestSettingsPopup.testHelpers";

beforeAll(() => {
  client.setConfig({ baseUrl: "http://localhost" });
  Element.prototype.hasPointerCapture ??= () => false;
  Element.prototype.setPointerCapture ??= () => {};
  Element.prototype.releasePointerCapture ??= () => {};
  Element.prototype.scrollIntoView ??= () => {};
});
afterEach(() => vi.restoreAllMocks());

describe("Review request production settings", () => {
  it("keeps installed settings editable with the flag off and saves model, reasoning, and cleared fallback after a failed commit", async () => {
    const user = userEvent.setup();
    const settings = renderSettings({ failSave: true });
    const fallback = await screen.findByRole("combobox", { name: "Fallback reviewer" });
    await waitFor(() => expect(screen.getByTestId("review-request-settings-save")).toBeEnabled());
    expect(fallback).toHaveTextContent("saved-reviewer");
    expect(screen.getByRole("tab", { name: "Configuration" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Runs" })).toBeInTheDocument();
    expect(screen.queryByRole("textbox", { name: "Additional instructions" })).not.toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "Publish PR check" })).not.toBeInTheDocument();
    await user.click(fallback);
    await user.click(screen.getByRole("option", { name: "None" }));
    await waitForPopupDismissal();
    await user.click(screen.getByTestId("field-model-hosted-model"));
    await user.hover(screen.getByTestId("field-model-hosted-model-list"));
    fireEvent.click(await screen.findByRole("menuitem", { name: "openai/gpt-5.4" }));
    await user.hover(screen.getByTestId("field-model-hosted-thinking"));
    fireEvent.click(await screen.findByRole("menuitem", { name: "Medium" }));
    await user.keyboard("{Escape}");
    await waitForPopupDismissal();
    await user.click(screen.getByTestId("review-request-settings-save"));
    expect(await screen.findByRole("alert")).toHaveTextContent("Could not commit settings");
    expect(settings.onClose).not.toHaveBeenCalled();
    expect(screen.getByRole("combobox", { name: "Fallback reviewer" })).toHaveTextContent("None");
    await user.click(screen.getByTestId("review-request-settings-save"));
    await waitFor(() => expect(settings.onClose).toHaveBeenCalledTimes(1));
    expect(settings.stored().spec?.nodes?.find((node) => node.id === "select-reviewers")?.configuration).toMatchObject({
      fallbackReviewer: "",
      model: "hosted::openai::gpt-5.4",
      thinkingLevel: "medium",
    });
    settings.reload();
    expect(await screen.findByRole("combobox", { name: "Fallback reviewer" })).toHaveTextContent("None");
    expect(screen.getByTestId("field-model-hosted-model")).toHaveTextContent("gpt-5.4");
    expect(screen.getByTestId("field-model-hosted-model")).toHaveTextContent("Medium");
  });

  it("combines workspace repository reviewers without duplicates and preserves the saved fallback on lookup failure", async () => {
    const user = userEvent.setup();
    const settings = renderSettings();
    await screen.findByRole("combobox", { name: "Fallback reviewer" });
    await waitFor(() => expect(settings.reviewerRequests).toContain("acme/second"));
    expect(new Set(settings.reviewerRequests)).toEqual(new Set(["acme/main", "acme/second"]));
    await user.click(screen.getByRole("combobox", { name: "Fallback reviewer" }));
    expect(screen.getAllByRole("option").filter((option) => option.textContent?.toLowerCase() === "alex")).toHaveLength(
      1,
    );
    await user.click(screen.getByRole("option", { name: "sam" }));
    await waitForPopupDismissal();
    await user.click(screen.getByTestId("review-request-settings-save"));
    await waitFor(() => expect(settings.onClose).toHaveBeenCalledTimes(1));
    expect(
      settings.stored().spec?.nodes?.find((node) => node.id === "select-reviewers")?.configuration?.fallbackReviewer,
    ).toBe("sam");
    settings.view.unmount();
    settings.fetchMock.mockRestore();
    renderSettings({ failReviewers: true });
    expect(await screen.findByRole("combobox", { name: "Fallback reviewer" })).toHaveTextContent("saved-reviewer");
    expect(await screen.findByRole("alert")).toHaveTextContent("Could not load GitHub reviewers");
    expect(screen.getByRole("button", { name: "Try again" })).toBeEnabled();
  });

  it("pauses new reports without publishing unsaved fallback changes", async () => {
    const user = userEvent.setup();
    const settings = renderSettings();
    await user.click(await screen.findByRole("combobox", { name: "Fallback reviewer" }));
    await user.click(screen.getByRole("option", { name: "None" }));
    await waitForPopupDismissal();
    await user.click(screen.getByRole("button", { name: "Review request menu" }));
    await user.click(screen.getByRole("menuitem", { name: "Pause Review request" }));
    await waitFor(() =>
      expect(
        settings.stored().spec?.nodes?.find((node) => node.id === "review-request-trigger")?.configuration?.enabled,
      ).toBe(false),
    );
    expect(
      settings.stored().spec?.nodes?.find((node) => node.id === "select-reviewers")?.configuration?.fallbackReviewer,
    ).toBe("saved-reviewer");
    expect(
      within(screen.getByRole("region", { name: "Review request configuration" })).getByRole("status"),
    ).toHaveTextContent("paused");
  });

  it("shows the applied decision and retries selection from Runs after a retry error", async () => {
    const user = userEvent.setup();
    const settings = renderSettings({ failRetry: true, initialPath: "/?automationTab=runs&automationRun=run-1" });
    const summary = await screen.findByRole("region", { name: "Review decision" });
    expect(summary).toHaveTextContent("Ownership review is required for a cautious report.");
    expect(summary).toHaveTextContent("sam: Owns the changed package.");
    expect(summary).not.toHaveTextContent("Agent decision before application.");
    await user.click(screen.getByRole("button", { name: "Retry run" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Could not retry selection");
    await user.click(screen.getByRole("button", { name: "Retry run" }));
    await waitFor(() => expect(settings.retryRequests).toHaveLength(2));
    await waitFor(() => expect(screen.queryByRole("region", { name: "Review decision" })).not.toBeInTheDocument());
  });
});
