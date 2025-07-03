interface StoryData {
    viewMode?: string;
    storyId?: string;
    refId?: string;
}
declare const parsePath: (path: string | undefined) => StoryData;
interface Args {
    [key: string]: any;
}
declare const DEEPLY_EQUAL: unique symbol;
declare const deepDiff: (value: any, update: any) => any;
declare const buildArgsParam: (initialArgs: Args | undefined, args: Args) => string;
interface Query {
    [key: string]: any;
}
declare const queryFromString: (s?: string) => Query;
declare const queryFromLocation: (location: Partial<Location>) => Query;
declare const stringifyQuery: (query: Query) => string;
type Match = {
    path: string;
};
declare const getMatch: (current: string, target: string | RegExp, startsWith?: any) => Match | null;

export { DEEPLY_EQUAL, StoryData, buildArgsParam, deepDiff, getMatch, parsePath, queryFromLocation, queryFromString, stringifyQuery };
