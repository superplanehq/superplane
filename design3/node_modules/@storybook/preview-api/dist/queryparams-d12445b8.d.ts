import * as qs from 'qs';

declare const getQueryParams: () => qs.ParsedQs;
declare const getQueryParam: (key: string) => string | string[] | qs.ParsedQs | qs.ParsedQs[] | undefined;

export { getQueryParams as a, getQueryParam as g };
