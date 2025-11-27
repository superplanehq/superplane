import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import "./index.css";
import App from "./App.tsx";
import { setupApiInterceptor } from "./api-client/client.ts";

// Setup the API interceptor to handle 401 responses
setupApiInterceptor();

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
