const UNTITLED_APP_NAME_PATTERN = /^Untitled App (\d+)$/;

export function generateUntitledAppName(existingNames: string[]): string {
  const usedNumbers = existingNames
    .map((name) => {
      const match = name.match(UNTITLED_APP_NAME_PATTERN);
      return match ? Number.parseInt(match[1], 10) : null;
    })
    .filter((number): number is number => number !== null && !Number.isNaN(number));

  const nextNumber = usedNumbers.length > 0 ? Math.max(...usedNumbers) + 1 : 1;
  return `Untitled App ${nextNumber}`;
}
