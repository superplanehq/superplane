import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import App from "./App.tsx";
import { setupApiInterceptor } from "./lib/api-interceptor.ts";
import { Sentry } from "./sentry.ts";
import { ErrorPage } from "./components/ErrorPage.tsx";

// Setup the API interceptor to handle 401 responses
setupApiInterceptor();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Sentry.ErrorBoundary fallback={<ErrorPage />}>
      <App />
    </Sentry.ErrorBoundary>
  </StrictMode>,
);
