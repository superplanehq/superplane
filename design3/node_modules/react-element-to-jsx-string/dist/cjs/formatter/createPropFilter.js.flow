/* @flow */

export default function createPropFilter(
  props: {},
  filter: string[] | ((any, string) => boolean)
) {
  if (Array.isArray(filter)) {
    return (key: string) => filter.indexOf(key) === -1;
  } else {
    return (key: string) => filter(props[key], key);
  }
}
