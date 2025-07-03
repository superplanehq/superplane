import { Options } from '@storybook/types';
import { Channel } from '@storybook/channels';

declare const experimental_serverChannel: (channel: Channel, options: Options) => Promise<Channel>;

export { experimental_serverChannel };
