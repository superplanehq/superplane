import type { Meta, StoryObj } from "@storybook/react-vite";
import { http, HttpResponse } from "msw";
import { MemoryRouter } from "react-router";
import { userEvent, within } from "storybook/test";
import { AccountContext } from "@/contexts/accountContextState";
import { Login } from "./Login";

const meta = {
  title: "Auth/Login branding",
  component: Login,
  parameters: {
    layout: "fullscreen",
    msw: {
      handlers: [
        http.get("/auth/config", () =>
          HttpResponse.json({
            providers: ["google", "github"],
            passwordLoginEnabled: false,
            signupEnabled: true,
            signupsBlockedByEnvironment: false,
            magicCodeEnabled: true,
          }),
        ),
        http.post("/auth/magic-code/request", () => new HttpResponse(null, { status: 200 })),
      ],
    },
  },
  decorators: [
    (Story) => (
      <AccountContext.Provider
        value={{ account: null, loading: false, setupRequired: false, refreshAccount: async () => undefined }}
      >
        <MemoryRouter initialEntries={["/login"]}>
          <Story />
        </MemoryRouter>
      </AccountContext.Provider>
    ),
  ],
} satisfies Meta<typeof Login>;
export default meta;
type Story = StoryObj<typeof meta>;
export const Email: Story = {};
export const Dark: Story = { globals: { theme: "dark" } };
export const Mobile: Story = { globals: { viewport: { value: "mobile1", isRotated: false } } };
export const EmailCode: Story = {
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);
    await userEvent.type(await canvas.findByPlaceholderText("you@example.com"), "you@example.com");
    await userEvent.click(canvas.getByRole("button", { name: "Continue with email" }));
    await canvas.findByRole("heading", { name: "Check your email" });
  },
};
export const EmailCodeDark: Story = { ...EmailCode, globals: { theme: "dark" } };
export const Error: Story = {
  parameters: { msw: { handlers: [http.get("/auth/config", () => new HttpResponse(null, { status: 500 }))] } },
};
