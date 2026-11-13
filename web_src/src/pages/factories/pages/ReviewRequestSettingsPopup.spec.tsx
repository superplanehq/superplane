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
  it("keeps legacy settings editable and explains the canvas upgrade before enabling checks", async () => {
    const user = userEvent.setup();
    const settings = renderSettings({ legacy: true });
    const instructions = await screen.findByRole("textbox", { name: "Additional instructions" });
    fireEvent.change(instructions, { target: { value: "Prefer the security owner." } });
    await user.click(screen.getByTestId("review-request-settings-save"));
    await waitFor(() => expect(settings.onClose).toHaveBeenCalledTimes(1));
    expect(savedConfiguration(settings, "select-reviewers")?.additionalInstructions).toBe("Prefer the security owner.");
    settings.reload();
    await user.click(await screen.findByRole("switch", { name: "Publish PR check" }));
    await user.click(screen.getByTestId("review-request-settings-save"));
    expect(await screen.findByRole("alert")).toHaveTextContent("admin reset");
    expect(savedConfiguration(settings, "select-reviewers")?.publishPRCheck).toBe(false);
  });
  it("saves agent settings and clears instructions through reload after a failed commit with the flag off", async () => {
    const user = userEvent.setup();
    const settings = renderSettings({ failSave: true });
    const fallback = await screen.findByRole("combobox", { name: "Fallback reviewer" });
    await waitFor(() => expect(screen.getByTestId("review-request-settings-save")).toBeEnabled());
    expect(fallback).toHaveTextContent("saved-reviewer");
    expect(screen.getByRole("tab", { name: "Configuration" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Runs" })).toBeInTheDocument();
    const instructions = screen.getByRole("textbox", { name: "Additional instructions" });
    expect(instructions).toHaveValue("");
    const value = "Prefer the security owner. Explain ownership using {{ example }} literally.";
    fireEvent.change(instructions, { target: { value } });
    fireEvent.blur(instructions);
    expect(
      settings.stored().spec?.nodes?.find((node) => node.id === "select-reviewers")?.configuration
        ?.additionalInstructions,
    ).toBeUndefined();
    const publishCheck = screen.getByRole("switch", { name: "Publish PR check" });
    expect(publishCheck).not.toBeChecked();
    await user.click(publishCheck);
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
    expect(instructions).toHaveValue(value);
    expect(screen.getByRole("combobox", { name: "Fallback reviewer" })).toHaveTextContent("None");
    await user.click(screen.getByTestId("review-request-settings-save"));
    await waitFor(() => expect(settings.onClose).toHaveBeenCalledTimes(1));
    expect(settings.stored().spec?.nodes?.find((node) => node.id === "select-reviewers")?.configuration).toMatchObject({
      fallbackReviewer: "",
      model: "hosted::openai::gpt-5.4",
      thinkingLevel: "medium",
      additionalInstructions: value,
      publishPRCheck: true,
      steps: [{ name: "Select reviewers", type: "prompt", prompt: "Managed reviewer policy" }],
    });
    settings.reload();
    expect(await screen.findByRole("combobox", { name: "Fallback reviewer" })).toHaveTextContent("None");
    expect(screen.getByTestId("field-model-hosted-model")).toHaveTextContent("gpt-5.4");
    expect(screen.getByTestId("field-model-hosted-model")).toHaveTextContent("Medium");
    expect(screen.getByRole("textbox", { name: "Additional instructions" })).toHaveValue(value);
    expect(screen.getByRole("switch", { name: "Publish PR check" })).toBeChecked();
    await user.click(screen.getByRole("switch", { name: "Publish PR check" }));
    for (const next of ["Prefer the API owner and explain the evidence.", ""]) {
      fireEvent.change(screen.getByRole("textbox", { name: "Additional instructions" }), { target: { value: next } });
      await user.click(screen.getByTestId("review-request-settings-save"));
      await waitFor(() =>
        expect(
          settings.stored().spec?.nodes?.find((node) => node.id === "select-reviewers")?.configuration
            ?.additionalInstructions,
        ).toBe(next),
      );
      settings.reload();
      expect(await screen.findByRole("textbox", { name: "Additional instructions" })).toHaveValue(next);
      expect(screen.getByRole("switch", { name: "Publish PR check" })).not.toBeChecked();
      expect(savedConfiguration(settings, "select-reviewers")?.publishPRCheck).toBe(false);
      expect(savedConfiguration(settings, "pr-check-pending")?.enabled).toBe(false);
    }
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
    fireEvent.change(screen.getByRole("textbox", { name: "Additional instructions" }), {
      target: { value: "Unsaved ownership preference." },
    });
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
      settings.stored().spec?.nodes?.find((node) => node.id === "select-reviewers")?.configuration
        ?.additionalInstructions,
    ).toBeUndefined();
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

function savedConfiguration(settings: ReturnType<typeof renderSettings>, nodeId: string) {
  return settings.stored().spec?.nodes?.find((node) => node.id === nodeId)?.configuration;
}
