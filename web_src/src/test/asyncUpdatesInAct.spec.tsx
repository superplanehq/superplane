import { QueryClient, QueryClientProvider, useQuery } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { useEffect, useState, type ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";

function RafLabel() {
  const [label, setLabel] = useState("pending");

  useEffect(() => {
    const frame = requestAnimationFrame(() => {
      setLabel("ready");
    });
    return () => cancelAnimationFrame(frame);
  }, []);

  return <span>{label}</span>;
}

function QueryLabel() {
  const query = useQuery({
    queryKey: ["async-updates-in-act"],
    queryFn: async () => "ready",
  });

  return <span>{query.data ?? "pending"}</span>;
}

function queryWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

function actWarningFrom(error: ReturnType<typeof vi.spyOn>) {
  return error.mock.calls.flat().some((argument) => typeof argument === "string" && argument.includes("not wrapped in act"));
}

describe("async updates in tests", () => {
  it("applies requestAnimationFrame state updates without an act warning", async () => {
    const error = vi.spyOn(console, "error").mockImplementation(() => undefined);

    render(<RafLabel />);

    expect(await screen.findByText("ready")).toBeInTheDocument();
    expect(actWarningFrom(error)).toBe(false);
    error.mockRestore();
  });

  it("applies query cache notifications without an act warning", async () => {
    const error = vi.spyOn(console, "error").mockImplementation(() => undefined);
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    render(<QueryLabel />, { wrapper: queryWrapper(queryClient) });

    expect(await screen.findByText("ready")).toBeInTheDocument();
    expect(actWarningFrom(error)).toBe(false);
    error.mockRestore();
    queryClient.clear();
  });
});
