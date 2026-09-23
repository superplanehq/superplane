import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router";

import { PriceBooks } from "./PriceBooks";

export const versions = [
  { version: "2026-09-09.1", effective_at: "2026-09-09T00:00:00Z", created_at: "2026-09-09T00:00:00Z" },
  { version: "2026-09-01.1", effective_at: "2026-09-01T00:00:00Z", created_at: "2026-09-01T00:00:00Z" },
  { version: "2026-08-31.1", effective_at: "2026-08-31T00:00:00Z", created_at: "2026-08-31T00:00:00Z" },
];

export const catalogFor = (version: string, matchKey: string, current = "2026-09-09.1") => ({
  current_version: current,
  version,
  effective_at: `${version.slice(0, 10)}T00:00:00Z`,
  created_at: `${version.slice(0, 10)}T00:00:00Z`,
  versions,
  models: [
    {
      provider: "openrouter",
      match_key: matchKey,
      match_mode: "exact",
      input_cents_per_million: 300,
      output_cents_per_million: 1500,
      cache_read_cents_per_million: 30,
      cache_write_cents_per_million: 375,
      reasoning_cents_per_million: 0,
      selected: true,
    },
  ],
  vms: [
    {
      match_key: "e1-large-amd64",
      match_mode: "exact",
      micros_per_second: 70,
    },
  ],
});

export const currentCatalog = catalogFor("2026-09-09.1", "claude-sonnet");
export const olderCatalog = catalogFor("2026-08-31.1", "older-model");
export const middleCatalog = catalogFor("2026-09-01.1", "middle-model");
export const savedCatalog = {
  ...catalogFor("2026-09-13.1", "claude-sonnet", "2026-09-13.1"),
  versions: [
    { version: "2026-09-13.1", effective_at: "2026-09-13T00:00:00Z", created_at: "2026-09-13T00:00:00Z" },
    ...versions,
  ],
};

export const jsonResponse = (body: unknown) =>
  new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });

export const renderPage = () =>
  render(
    <MemoryRouter initialEntries={["/admin/price-books"]}>
      <PriceBooks />
    </MemoryRouter>,
  );
