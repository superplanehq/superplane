import { P as PresetProperty, S as StorybookConfig } from './index-f49558df.js';
import 'file-system-cache';
import '@babel/core';
import 'http';
import '@storybook/builder-vite';
import '@joshwooding/vite-plugin-react-docgen-typescript';

declare const core: PresetProperty<'core'>;
declare const viteFinal: StorybookConfig['viteFinal'];

export { core, viteFinal };
