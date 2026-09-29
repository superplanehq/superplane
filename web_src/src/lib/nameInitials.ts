/**
 * Minimal local type for the `Intl.Segmenter` API used in this file.
 *
 * The project's `tsconfig.app.json` targets the `ES2020` lib, which does not
 * yet include `Intl.Segmenter` types (that only lands with newer lib targets
 * such as `ESNext`). The API is well supported at runtime in all browsers and
 * Node versions this project targets, so we describe just the surface we use
 * here instead of widening the whole project's `lib` setting.
 */
interface NameInitialsGraphemeSegment {
  segment: string;
}

interface NameInitialsSegmenter {
  segment(input: string): Iterable<NameInitialsGraphemeSegment>;
}

interface NameInitialsSegmenterConstructor {
  new (
    locales?: string | string[],
    options?: { granularity?: "grapheme" | "word" | "sentence" },
  ): NameInitialsSegmenter;
}

const Segmenter = (Intl as unknown as { Segmenter: NameInitialsSegmenterConstructor }).Segmenter;

const WORD_WITH_LETTER_OR_DIGIT = /[\p{L}\p{N}]/u;

const graphemeSegmenter = new Segmenter(undefined, { granularity: "grapheme" });

/** The first grapheme cluster of `value`, or "" if `value` is empty. */
function firstGrapheme(value: string): string {
  for (const { segment } of graphemeSegmenter.segment(value)) {
    return segment;
  }
  return "";
}

/**
 * One or two letters/digits derived from a display name, for use as avatar or
 * badge initials.
 *
 * Words made up only of symbols (for example emoji) are dropped, so a name
 * like "SuperPlane Prod 🚀" yields "SP" instead of a broken glyph: taking
 * `part[0]` of an emoji word returns a lone surrogate, which the browser
 * renders as a replacement character.
 *
 * Returns the first grapheme of the first remaining word, plus the first
 * grapheme of the last remaining word when more than one word is left.
 * Returns "" when no word contains a letter or digit.
 */
export function getNameInitials(name: string): string {
  const words = name.split(/\s+/).filter((word) => WORD_WITH_LETTER_OR_DIGIT.test(word));

  if (words.length === 0) {
    return "";
  }

  const first = firstGrapheme(words[0]);
  if (words.length === 1) {
    return first.toUpperCase();
  }

  const last = firstGrapheme(words[words.length - 1]);
  return `${first}${last}`.toUpperCase();
}
