import type { Preview } from "@storybook/react-vite";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { initialize, mswLoader } from "msw-storybook-addon";
import React from "react";
import "../src/App.css";
import "../src/index.css";

// Load Material Symbols font for icons
const link = document.createElement("link");
link.href = "https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined";
link.rel = "stylesheet";
document.head.appendChild(link);

// Start MSW. Only stories that declare `parameters.msw.handlers` mock the
// network; everything else passes through untouched. Unhandled /api requests
// warn (so missing handlers surface during development) while static assets
// and other requests are ignored.
initialize({
  onUnhandledRequest: (request, print) => {
    if (new URL(request.url).pathname.startsWith("/api")) {
      print.warning();
    }
  },
});

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      staleTime: Infinity,
    },
  },
});

const preview: Preview = {
  loaders: [mswLoader],
  decorators: [
    (Story) => (
      <QueryClientProvider client={queryClient}>
        <Story />
      </QueryClientProvider>
    ),
  ],
  parameters: {
    options: {
      storySort: {
        method: "alphabetical",
        locales: "en-US",
      },
    },
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
    backgrounds: {
      default: "light",
      values: [
        {
          name: "light",
          value: "#ffffff",
        },
        {
          name: "dark",
          value: "#1a1a1a",
        },
      ],
    },
  },
};

export default preview;
