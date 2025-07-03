/* @flow */

const isKeyOrRefProps = (propName: string) => ['key', 'ref'].includes(propName);

export default (shouldSortUserProps: boolean) => (
  props: string[]
): string[] => {
  const haveKeyProp = props.includes('key');
  const haveRefProp = props.includes('ref');

  const userPropsOnly = props.filter(oneProp => !isKeyOrRefProps(oneProp));

  const sortedProps = shouldSortUserProps
    ? [...userPropsOnly.sort()] // We use basic lexical order
    : [...userPropsOnly];

  if (haveRefProp) {
    sortedProps.unshift('ref');
  }

  if (haveKeyProp) {
    sortedProps.unshift('key');
  }

  return sortedProps;
};
