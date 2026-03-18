import {
  DEFAULT_OUTPUT_CHANNEL,
  type ComponentDefinition,
} from "@superplanehq/sdk";

export const helloWorld = {
  name: "helloWorld",
  label: "Hello World",
  description: "Emits a hello world message",
  icon: "message-circle",
  color: "green",
  configuration: [],
  outputChannels: [DEFAULT_OUTPUT_CHANNEL],
  async execute({ context }) {
    await context.executionState.emit(
      DEFAULT_OUTPUT_CHANNEL.name,
      "helloWorld.message",
      [
        {
          message: "Hello, world!",
        },
      ],
    );
  },
} satisfies ComponentDefinition;
