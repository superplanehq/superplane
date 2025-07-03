/// <reference types="node" />
/// <reference types="node" />
import fs from 'fs';
import { FakeFS } from '../FakeFS';
import { NativePath } from '../path';
export declare function patchFs(patchedFs: typeof fs, fakeFs: FakeFS<NativePath>): void;
export declare function extendFs(realFs: typeof fs, fakeFs: FakeFS<NativePath>): typeof fs;
