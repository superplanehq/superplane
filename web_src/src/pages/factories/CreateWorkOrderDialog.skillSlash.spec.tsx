import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "bun:test";

import {
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY,
  factoryWithPlanning,
} from "./__fixtures__/factoryPageResponses";
import { CreateWorkOrderDialog } from "./CreateWorkOrderDialog";
import { FactoriesLayoutContext } from "./layout/factoriesLayoutContext";

vi.mock("@/hooks/useFactoryData", () => ({
  useCreateWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useDispatchWorkOrder: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({ data: null }),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
}));

vi.mock("@/hooks/useSkillSlashCandidates", () => ({
  useSkillSlashCandidates: (
    _organizationId: string | undefined,
    _factoryId: string | undefined,
    filter: string,
    enabled: boolean,
  ) => {
    if (!enabled) {
      return [];
    }
    const all = [{ id: "1", command: "oypirate", title: "Oy Pirate!", description: "Talk like a pirate." }];
    const needle = filter.trim().toLowerCase();
    return needle ? all.filter((candidate) => candidate.command.includes(needle)) : all;
  },
}));

const emptyRect = {
  x: 0,
  y: 0,
  width: 0,
  height: 0,
  top: 0,
  right: 0,
  bottom: 0,
  left: 0,
  toJSON() {
    return this;
  },
};
const emptyRects = {
  item: () => null,
  length: 0,
  [Symbol.iterator]: function* () {},
};

beforeAll(() => {
  document.elementFromPoint = () => null;
  stubClientRects(Range.prototype);
  stubClientRects(Element.prototype);
  stubClientRects(Text.prototype);
});

function stubClientRects(target: object) {
  Object.defineProperty(target, "getBoundingClientRect", {
    configurable: true,
    value: () => emptyRect,
  });
  Object.defineProperty(target, "getClientRects", {
    configurable: true,
    value: () => emptyRects,
  });
}

afterEach(async () => {
  cleanup();
  await act(async () => {
    await new Promise((resolve) => setTimeout(resolve, 5));
  });
});

describe("CreateWorkOrderDialog skill slash", () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it("inserts a skill command when Planning is off", async () => {
    const user = userEvent.setup();
    const factory = factoryWithPlanning(REFUND_FACTORY, { enabled: false, clarity: true, confidence: true });

    render(
      <FactoriesLayoutContext.Provider
        value={{
          organizationId: "org-1",
          factoryId: PRIMARY_FACTORY_ID,
          factoryKey: PRIMARY_FACTORY_KEY,
          factory,
          factories: [factory],
          openCreateWorkOrder: vi.fn(),
        }}
      >
        <CreateWorkOrderDialog open onClose={vi.fn()} onCreated={vi.fn()} />
      </FactoriesLayoutContext.Provider>,
    );

    const input = await screen.findByTestId("work-order-description-input");
    await user.click(input);
    await user.keyboard("/oy");

    expect(await screen.findByTestId("skill-slash-menu")).toBeInTheDocument();
    await user.click(screen.getByTestId("skill-slash-option-oypirate"));

    await waitFor(() => {
      expect(input).toHaveTextContent("/oypirate");
    });
  });
});
