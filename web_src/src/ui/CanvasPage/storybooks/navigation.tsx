export function isInStorybook() {
  return (
    typeof window !== "undefined" &&
    (window.location.pathname.includes("storybook") ||
      window.location.search.includes("path=/story/") ||
      (window.parent !== window && window.parent.location.pathname.includes("storybook")))
  );
}

export const navigateToStoryWithData = (storyId: string, data?: any) => {
  try {
    // Use parent window location for Storybook iframe navigation
    const targetWindow = window.parent !== window ? window.parent : window;
    let newUrl = `${targetWindow.location.origin}${targetWindow.location.pathname}?path=/story/${storyId}`;

    // Add data as query parameters for Storybook
    if (data) {
      const encodedData = encodeURIComponent(JSON.stringify(data));
      newUrl += `&args=nodeData:${encodedData}`;
    }

    // Navigate using the correct window
    targetWindow.location.href = newUrl;
  } catch (error) {
    console.error("❌ Navigation failed:", error);
    // Ultimate fallback - try direct URL construction
    try {
      const fallbackUrl = `${window.location.protocol}//${window.location.host}${window.location.pathname}?path=/story/${storyId}`;
      if (window.top?.location) {
        window.top.location.href = fallbackUrl;
      }
    } catch (fallbackError) {
      console.error("❌ Fallback also failed:", fallbackError);
    }
  }
};

export const navigateToStory = (storyId: string) => {
  navigateToStoryWithData(storyId);
};

export const getStorybookData = () => {
  if (typeof window === "undefined") return null;

  try {
    const urlParams = new URLSearchParams(window.location.search);

    const args = urlParams.get("args");

    if (args) {
      // Parse args string like "nodeData:encodedJson"
      const nodeDataMatch = args.match(/nodeData:([^&]+)/);

      if (nodeDataMatch) {
        const decodedData = decodeURIComponent(nodeDataMatch[1]);
        const parsedData = JSON.parse(decodedData);
        return parsedData;
      }
    }
  } catch (error) {
    console.error("❌ Failed to parse Storybook data:", error);
  }

  return null;
};

export const handleNodeExpand = (nodeId: string, nodeData: unknown) => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const nodeTitle = (nodeData as any).label;

  const executionData = {
    title: nodeTitle,
    parentWorkflow: "Simple Deployment",
    nodeId: nodeId,
    timestamp: Date.now(),
  };

  navigateToStoryWithData("pages-canvaspage--blueprint-execution-page", executionData);
  return;
};
