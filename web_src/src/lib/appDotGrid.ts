const MIN_VISIBLE_DOTS = 12;
const MAX_VISIBLE_DOTS = 24;
const DEFAULT_THRESHOLD = 0.45;

function hashString(value: string): number {
  let hash = 2166136261;

  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index);
    hash = Math.imul(hash, 16777619);
  }

  return hash >>> 0;
}

function createSeededRandom(seed: number): () => number {
  let state = seed >>> 0;

  return () => {
    state = Math.imul(state ^ (state >>> 16), 2246822507);
    state = Math.imul(state ^ (state >>> 13), 3266489909);
    state ^= state >>> 16;
    return (state >>> 0) / 4294967296;
  };
}

function countVisibleDots(dots: boolean[]): number {
  return dots.reduce((count, visible) => count + (visible ? 1 : 0), 0);
}

function generateDots(random: () => number, total: number, threshold: number): boolean[] {
  return Array.from({ length: total }, () => random() > threshold);
}

export function generateAppDotGrid(seed: string, gridSize = 6): boolean[] {
  const total = gridSize * gridSize;
  const baseHash = hashString(seed);

  for (let attempt = 0; attempt < 8; attempt += 1) {
    const random = createSeededRandom(baseHash + attempt * 1013904223);
    const threshold = DEFAULT_THRESHOLD + attempt * 0.03;
    const dots = generateDots(random, total, threshold);
    const visibleCount = countVisibleDots(dots);

    if (visibleCount >= MIN_VISIBLE_DOTS && visibleCount <= MAX_VISIBLE_DOTS) {
      return dots;
    }
  }

  const random = createSeededRandom(baseHash);
  const targetVisible = MIN_VISIBLE_DOTS + (baseHash % (MAX_VISIBLE_DOTS - MIN_VISIBLE_DOTS + 1));
  const dots = Array.from({ length: total }, () => false);
  const indices = Array.from({ length: total }, (_, index) => index);

  for (let index = indices.length - 1; index > 0; index -= 1) {
    const swapIndex = Math.floor(random() * (index + 1));
    [indices[index], indices[swapIndex]] = [indices[swapIndex], indices[index]];
  }

  for (let index = 0; index < targetVisible; index += 1) {
    dots[indices[index]] = true;
  }

  return dots;
}
