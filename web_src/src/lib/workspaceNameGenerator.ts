/**
 * Random starter names for workspaces.
 * Tone: craft + motion — a mighty works that moves small engineering work forward.
 * Format: Title Case words with a space (e.g. "Tempered Loom"), unlike app `adj-noun` slugs.
 */
const adjectives = [
  "Tempered",
  "Keen",
  "Braced",
  "Proven",
  "Riveted",
  "Honed",
  "Sturdy",
  "Swift",
  "Forged",
  "Seasoned",
  "Hammered",
  "True",
  "Solid",
  "Bright",
  "Steady",
  "Resolute",
  "Mighty",
  "Ironclad",
  "Vaulted",
  "Granite",
];

const nouns = [
  "Anvil",
  "Loom",
  "Lathe",
  "Relay",
  "Gantry",
  "Crucible",
  "Foundry",
  "Scaffold",
  "Bellows",
  "Hearth",
  "Kiln",
  "Vise",
  "Forge",
  "Mill",
  "Works",
  "Yard",
  "Engine",
  "Bench",
  "Dock",
  "Bay",
];

function pick(items: string[]): string {
  return items[Math.floor(Math.random() * items.length)];
}

export function generateWorkspaceName(): string {
  return `${pick(adjectives)} ${pick(nouns)}`;
}
