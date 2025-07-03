import type { Options } from './../options';

function noRefCheck() {}

export const inlineFunction = (fn: any): string =>
  fn
    .toString()
    .split('\n')
    .map(line => line.trim())
    .join('');

export const preserveFunctionLineBreak = (fn: any): string => fn.toString();

const defaultFunctionValue = inlineFunction;

export default (fn: Function, options: Options): string => {
  const { functionValue = defaultFunctionValue, showFunctions } = options;
  if (!showFunctions && functionValue === defaultFunctionValue) {
    return functionValue(noRefCheck);
  }

  return functionValue(fn);
};
