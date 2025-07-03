/* eslint-disable no-underscore-dangle */
import { global } from '@storybook/global';

export const setJSONDoc = (jsondoc) => {
  global.__EMBER_GENERATED_DOC_JSON__ = jsondoc;
};
