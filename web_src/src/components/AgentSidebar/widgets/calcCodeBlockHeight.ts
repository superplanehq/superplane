const CODE_LINE_HEIGHT_PX = 19;
const CODE_VERTICAL_PADDING_PX = 16;
export const CODE_BLOCK_MAX_HEIGHT_PX = 250;

export function calcCodeBlockHeight(code: string, maxPx = CODE_BLOCK_MAX_HEIGHT_PX): number {
  const lineCount = Math.max(code.split("\n").length, 1);
  return Math.min(lineCount * CODE_LINE_HEIGHT_PX + CODE_VERTICAL_PADDING_PX, maxPx);
}
