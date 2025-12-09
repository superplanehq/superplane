import type { StorybookConfig } from "@storybook/react-vite";
import * as path from "path";

const config: StorybookConfig = {
  stories: ["../src/**/*.stories.@(js|jsx|mjs|ts|tsx)"],
  addons: ["@chromatic-com/storybook", "@storybook/addon-docs", "@storybook/addon-onboarding", "@storybook/addon-a11y"],
  framework: {
    name: "@storybook/react-vite",
    options: {},
  },
  viteFinal: async (config) => {
    // Remove the custom plugin that requires strictPort
    config.plugins = config.plugins?.filter(
      (plugin) =>
        !(plugin && typeof plugin === "object" && "name" in plugin && plugin.name === "set-hmr-port-from-port"),
    );

    // Configure server settings
    config.server = {
      ...config.server,
      strictPort: false,
    };

    // Add path aliases to match your main Vite config
    config.resolve = {
      ...config.resolve,
      alias: {
        ...config.resolve?.alias,
        "@/canvas": path.resolve(__dirname, "../src/pages/canvas"),
        "@": path.resolve(__dirname, "../src"),
      },
    };

    return config;
  },
};
export default config;
