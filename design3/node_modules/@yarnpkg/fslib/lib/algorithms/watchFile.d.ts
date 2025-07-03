import { FakeFS, WatchFileOptions, WatchFileCallback } from '../FakeFS';
import { Path } from '../path';
import { CustomStatWatcher } from './watchFile/CustomStatWatcher';
export declare function watchFile<P extends Path>(fakeFs: FakeFS<P>, path: P, a: WatchFileOptions | WatchFileCallback, b?: WatchFileCallback): CustomStatWatcher<P>;
export declare function unwatchFile<P extends Path>(fakeFs: FakeFS<P>, path: P, cb?: WatchFileCallback): void;
export declare function unwatchAllFiles(fakeFs: FakeFS<Path>): void;
