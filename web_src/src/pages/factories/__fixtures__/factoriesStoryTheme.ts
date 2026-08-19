import type { Decorator } from "@storybook/react-vite";
import { createElement } from "react";
import { FactoriesThemeStoryShell } from "./factoriesStoryDecorators";

/**
 * Storybook decorator that mounts every story inside `FactoriesThemeStoryShell`,
 * so the v3 tokens from `App.css` apply while the story is rendered.
 */
export const withFactoriesTheme: Decorator = (Story) =>
  createElement(FactoriesThemeStoryShell, null, createElement(Story));
