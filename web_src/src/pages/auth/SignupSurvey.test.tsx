import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import SignupSurvey from "./SignupSurvey";

beforeEach(() => {
  global.fetch = vi.fn();
  Object.defineProperty(window, "location", {
    value: { href: "/", assign: vi.fn() },
    writable: true,
  });
});

function renderPage() {
  return render(
    <MemoryRouter>
      <SignupSurvey />
    </MemoryRouter>,
  );
}

describe("SignupSurvey", () => {
  it("disables Continue until a source channel is chosen", async () => {
    renderPage();
    const cont = screen.getByRole("button", { name: /continue/i });
    expect(cont).toBeDisabled();

    await userEvent.click(screen.getByLabelText(/search engine/i));
    expect(cont).toBeEnabled();
  });

  it("always enables Skip for now", () => {
    renderPage();
    expect(screen.getByRole("button", { name: /skip for now/i })).toBeEnabled();
  });

  it("submits the expected payload on Continue", async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 204 });
    global.fetch = fetchMock;

    renderPage();
    await userEvent.click(screen.getByLabelText(/search engine/i));
    await userEvent.click(screen.getByRole("button", { name: /^engineer \//i }));
    await userEvent.type(screen.getByLabelText(/what do you want to use superplane/i), "deploy ML jobs");
    await userEvent.click(screen.getByRole("button", { name: /continue/i }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalled());
    const [url, init] = fetchMock.mock.calls[0];
    expect(url).toBe("/signup-survey");
    expect(init.method).toBe("POST");
    const body = JSON.parse(init.body);
    expect(body).toEqual({
      skipped: false,
      source_channel: "search",
      role: "engineer",
      use_case: "deploy ML jobs",
    });
  });

  it("submits skipped=true on Skip for now", async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 204 });
    global.fetch = fetchMock;

    renderPage();
    await userEvent.click(screen.getByRole("button", { name: /skip for now/i }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalled());
    const [, init] = fetchMock.mock.calls[0];
    expect(JSON.parse(init.body)).toEqual({ skipped: true });
  });

  it("shows an error banner when the API fails and keeps buttons enabled", async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: false, status: 500 });
    global.fetch = fetchMock;

    renderPage();
    await userEvent.click(screen.getByLabelText(/search engine/i));
    await userEvent.click(screen.getByRole("button", { name: /continue/i }));

    expect(await screen.findByText(/we couldn.t save your answers/i)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /continue/i })).toBeEnabled();
    expect(screen.getByRole("button", { name: /skip for now/i })).toBeEnabled();
  });

  it("reveals the Other input when Other is selected", async () => {
    renderPage();
    expect(screen.queryByPlaceholderText(/tell us where/i)).not.toBeInTheDocument();
    await userEvent.click(screen.getByLabelText(/^other$/i));
    expect(screen.getByPlaceholderText(/tell us where/i)).toBeInTheDocument();
  });
});
