import { createPlanelet } from "@superplane/planelet-sdk";

const planelet = createPlanelet({
  id: "quotes",
  label: "Random Quotes",
  icon: "quote",
  iconUrl: "https://example.com/quote.svg",
  description: "Get random quotes and generate greetings",
});

const quotes = [
  {
    text: "The only way to do great work is to love what you do.",
    author: "Steve Jobs",
  },
  {
    text: "Innovation distinguishes between a leader and a follower.",
    author: "Steve Jobs",
  },
  { text: "Stay hungry, stay foolish.", author: "Steve Jobs" },
  { text: "Move fast and break things.", author: "Mark Zuckerberg" },
  {
    text: "The best way to predict the future is to invent it.",
    author: "Alan Kay",
  },
  { text: "Talk is cheap. Show me the code.", author: "Linus Torvalds" },
];

planelet.action("get-quote", {
  label: "Get Random Quote",
  description: "Returns a random inspirational quote",
  parameters: {
    category: {
      label: "Category",
      type: "select",
      description: "Filter quotes by category",
      required: false,
      options: [
        { label: "All", value: "all" },
        { label: "Innovation", value: "innovation" },
        { label: "Motivation", value: "motivation" },
      ],
    },
  },
  execute: async () => {
    const idx = Math.floor(Math.random() * quotes.length);
    const quote = quotes[idx];
    return {
      quote: quote.text,
      author: quote.author,
      index: idx,
    };
  },
});

planelet.action("greet", {
  label: "Generate Greeting",
  description: "Generate a personalized greeting message",
  parameters: {
    name: {
      label: "Name",
      type: "string",
      description: "Name of the person to greet",
      required: true,
    },
    style: {
      label: "Style",
      type: "select",
      description: "Greeting style",
      required: true,
      options: [
        { label: "Formal", value: "formal" },
        { label: "Casual", value: "casual" },
        { label: "Enthusiastic", value: "enthusiastic" },
      ],
    },
  },
  execute: async ({ parameters }) => {
    const name = parameters.name as string;
    const style = parameters.style as string;

    const greetings: Record<string, string> = {
      formal: `Good day, ${name}. It is a pleasure to make your acquaintance.`,
      casual: `Hey ${name}, what's up?`,
      enthusiastic: `OMG ${name}!!! SO GREAT to see you!`,
    };

    return {
      greeting: greetings[style] ?? greetings.casual,
      style,
      recipient: name,
    };
  },
});

planelet.trigger("quote-created", {
  label: "Quote Created",
  description: "Demo webhook trigger that normalizes incoming quote events",
  parameters: {
    workspaceId: {
      label: "Workspace ID",
      type: "string",
      required: true,
    },
  },
  setup: async ({ parameters, webhook }) => {
    return {
      providerWebhookId: "demo-webhook",
      workspaceId: parameters.workspaceId,
      webhookUrl: webhook.url,
    };
  },
  cleanup: async ({ metadata }) => {
    console.log("Cleaning up webhook", metadata?.providerWebhookId);
  },
  handleWebhook: async ({ request, metadata }) => {
    const rawBody = Buffer.from(request.rawBodyBase64, "base64").toString(
      "utf8",
    );
    const body = rawBody ? JSON.parse(rawBody) : {};

    return {
      eventType: "quote.created",
      payload: {
        providerWebhookId: metadata?.providerWebhookId,
        quote: body.quote,
        receivedMethod: request.method,
      },
      response: {
        status: 200,
        body: "ok",
      },
    };
  },
});

planelet.listen(3001);
