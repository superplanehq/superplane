import { Options } from 'got';
import { ExtractOptions } from 'tar-fs';

interface DownloadOptions {
    url: string;
    gotOpts?: Partial<Omit<Options, 'agent' | 'isStream'>>;
    extractOpts?: ExtractOptions;
    dir: string;
}
declare function download({ url, gotOpts, extractOpts, dir }: DownloadOptions): Promise<void>;

export { download as default };
