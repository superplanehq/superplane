import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import App from "./App.tsx";
import { setupApiInterceptor } from "./api-client/client.ts";
import { Sentry } from "./sentry.ts";

function ErrorFallback() {
  return <div>Something went wrong. Please refresh the page.</div>;
}

// Setup the API interceptor to handle 401 responses
setupApiInterceptor();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Sentry.ErrorBoundary fallback={<ErrorFallback />}>
      <App />
    </Sentry.ErrorBoundary>
  </StrictMode>,
);
