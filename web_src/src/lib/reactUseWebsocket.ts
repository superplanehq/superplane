import useWebSocketImport from "react-use-websocket";

type UseWebSocketHook = typeof useWebSocketImport;

// Vite 8 dev pre-bundling can expose the whole CJS module instead of unwrapping default.
function resolveUseWebSocket(imported: UseWebSocketHook | { default: UseWebSocketHook }): UseWebSocketHook {
  return typeof imported === "function" ? imported : imported.default;
}

export const useWebSocket = resolveUseWebSocket(useWebSocketImport as UseWebSocketHook | { default: UseWebSocketHook });
