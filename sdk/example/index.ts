import { createPlugin } from "@superplane/plugin-sdk";

const plugin = createPlugin({
  name: "quotes",
  label: "Random Quotes",
  icon: "quote",
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

plugin.action("get-quote", {
  label: "Get Random Quote",
  description: "Returns a random inspirational quote",
  fields: {
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

plugin.action("greet", {
  label: "Generate Greeting",
  description: "Generate a personalized greeting message",
  fields: {
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
  execute: async (params) => {
    const name = params.name as string;
    const style = params.style as string;

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

plugin.listen(3001);
