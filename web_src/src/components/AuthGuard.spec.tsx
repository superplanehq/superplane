import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { describe, expect, it, vi, beforeEach } from "vitest";
import AuthGuard from "@/components/AuthGuard";

const useAccountMock = vi.fn();

vi.mock("@/contexts/AccountContext", () => ({
  useAccount: () => useAccountMock(),
}));

describe("AuthGuard", () => {
  beforeEach(() => {
    useAccountMock.mockReset();
  });

  it("redirects unauthenticated users without triggering the render-phase navigate warning", async () => {
    useAccountMock.mockReturnValue({
      account: null,
      loading: false,
    });

    const consoleWarnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    render(
      <MemoryRouter initialEntries={["/private?tab=details"]}>
        <Routes>
          <Route
            path="/private"
            element={
              <AuthGuard>
                <div>secret</div>
              </AuthGuard>
            }
          />
          <Route path="/login" element={<div>login page</div>} />
        </Routes>
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(screen.getByText("login page")).toBeInTheDocument();
    });

    expect(consoleWarnSpy).not.toHaveBeenCalledWith(
      expect.stringContaining("You should call navigate() in a React.useEffect()"),
    );
    expect(consoleErrorSpy).not.toHaveBeenCalledWith(
      expect.stringContaining("You should call navigate() in a React.useEffect()"),
    );

    consoleWarnSpy.mockRestore();
    consoleErrorSpy.mockRestore();
  });

  it("renders protected content for authenticated users", () => {
    useAccountMock.mockReturnValue({
      account: { id: "acc-1" },
      loading: false,
    });

    render(
      <MemoryRouter>
        <AuthGuard>
          <div>secret</div>
        </AuthGuard>
      </MemoryRouter>,
    );

    expect(screen.getByText("secret")).toBeInTheDocument();
  });
});
