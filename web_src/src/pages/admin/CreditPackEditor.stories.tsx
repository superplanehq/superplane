import type { Meta, StoryObj } from "@storybook/react-vite";
import { delay, http, HttpResponse } from "msw";
import { CreditPackEditor } from "./CreditPackEditor";

const meta = {
  title: "Admin/CreditPackEditor",
  component: CreditPackEditor,
  parameters: {
    msw: {
      handlers: [
        http.get("/admin/api/polar/products", () =>
          HttpResponse.json([
            { id: "prod_starter", name: "Starter credits" },
            { id: "prod_custom", name: "Custom credits" },
          ]),
        ),
      ],
    },
  },
  args: {
    pack: null,
    onSave: async (pack) => {
      console.log("Save credit pack", pack);
    },
    onCancel: () => console.log("Cancel credit pack"),
  },
} satisfies Meta<typeof CreditPackEditor>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Create: Story = {};

export const Linked: Story = {
  args: {
    pack: {
      id: "pack-starter",
      name: "Starter credits",
      amount_cents: 2500,
      enabled: true,
      polar_product_id: "prod_starter",
    },
  },
};

export const CustomAmount: Story = {
  args: {
    pack: {
      id: "pack-custom",
      name: "Custom credits",
      amount_cents: 0,
      enabled: true,
      polar_product_id: "prod_custom",
    },
  },
};

export const SaveError: Story = {
  args: {
    ...Linked.args,
    onSave: async () => {
      throw new Error("Could not save the credit pack. Try again.");
    },
  },
};

export const ProductsLoading: Story = {
  parameters: {
    msw: {
      handlers: [
        http.get("/admin/api/polar/products", async () => {
          await delay("infinite");
          return HttpResponse.json([]);
        }),
      ],
    },
  },
};

export const ProductsUnavailable: Story = {
  args: Linked.args,
  parameters: {
    msw: {
      handlers: [
        http.get(
          "/admin/api/polar/products",
          () => new HttpResponse("Could not load Polar products. Try again.", { status: 502 }),
        ),
      ],
    },
  },
};

export const NoProducts: Story = {
  parameters: { msw: { handlers: [http.get("/admin/api/polar/products", () => HttpResponse.json([]))] } },
};
