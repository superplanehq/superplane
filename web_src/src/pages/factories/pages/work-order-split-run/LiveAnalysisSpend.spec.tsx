import { act, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";
import { LiveHeaderSpendProvider } from "./liveHeaderSpendContext";
import { LiveOwnerTimeCostRow } from "./LiveOwnerTimeCostRow";
import { PlanningHeaderSpendCollector } from "./PlanningHeaderSpendCollector";

describe("live analysis popup dollars", () => {
  it.each([
    {
      name: "hosted dollars increase before agent completion",
      hosted: true,
      costs: [undefined, undefined],
      dollars: ["0.18", "0.20"],
    },
    { name: "BYOK uses reported dollars", hosted: false, costs: [0.44, 0.55], dollars: ["0.44", "0.55"] },
    {
      name: "BYOK without reported dollars gets no estimate",
      hosted: false,
      costs: [undefined, undefined],
      dollars: ["0.00", "0.00"],
    },
  ])("$name", async ({ hosted, costs, dollars }) => {
    let writer!: ReadableStreamDefaultController<Uint8Array>;
    const body = new ReadableStream<Uint8Array>({
      start: (controller) => {
        writer = controller;
      },
    });
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            backend: "legacy",
            stream_url: "https://broker.example/logs",
            token: "test-token",
            hosted_pricing: hosted
              ? { rate: { Input: 300, Output: 1500, CacheRead: 30, CacheWrite: 375, Reasoning: 0 }, markup_bps: 2000 }
              : null,
          }),
          { headers: { "Content-Type": "application/json" } },
        ),
      )
      .mockResolvedValueOnce(new Response(body));
    const popup = render(
      <MemoryRouter>
        <LiveHeaderSpendProvider>
          <PlanningHeaderSpendCollector
            organizationId="org"
            saved={{ tokens: 0, cents: 0 }}
            view={{ canvasId: "canvas", executionId: "execution", messages: [], machineStatus: "running" }}
          />
          <LiveOwnerTimeCostRow
            fixture={{ owner: { id: "owner", name: "Alex", initials: "A" }, costUsd: "$0.00", tokensLabel: "0 tokens" }}
          />
        </LiveHeaderSpendProvider>
      </MemoryRouter>,
    );
    const sendTokens = (tokens: number, cost: number | undefined) =>
      writer.enqueue(
        new TextEncoder().encode(
          JSON.stringify({
            type: "turn",
            turn: 1,
            usage: { input_tokens: tokens, output_tokens: 0, ...(cost === undefined ? {} : { total_cost_usd: cost }) },
          }) + "\n",
        ),
      );
    try {
      await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
      await act(async () => sendTokens(50600, costs[0]));
      await waitFor(() =>
        expect(screen.getByTestId("popup-owner-time-cost")).toHaveTextContent(`$${dollars[0]} · 50.6k tokens`),
      );
      await act(async () => sendTokens(57200, costs[1]));
      await waitFor(() =>
        expect(screen.getByTestId("popup-owner-time-cost")).toHaveTextContent(`$${dollars[1]} · 57.2k tokens`),
      );
    } finally {
      popup.unmount();
      writer.close();
      fetchMock.mockRestore();
    }
  });
});
