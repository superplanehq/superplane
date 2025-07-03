import { g as globalApis } from './vendor-constants.538d9b49.js';
import { i as index } from './vendor-index.7646b3af.js';
import '@vitest/runner';
import './vendor-vi.6873a1c1.js';
import '@vitest/runner/utils';
import '@vitest/utils';
import './vendor-index.29282562.js';
import 'pathe';
import 'std-env';
import './vendor-global.97e4527c.js';
import 'chai';
import './vendor-_commonjsHelpers.7d1333e8.js';
import '@vitest/expect';
import '@vitest/snapshot';
import '@vitest/utils/error';
import './vendor-tasks.f9d75aed.js';
import '@vitest/utils/source-map';
import 'util';
import './vendor-date.6e993429.js';
import '@vitest/spy';
import './vendor-run-once.3e5ef7d7.js';

function registerApiGlobally() {
  globalApis.forEach((api) => {
    globalThis[api] = index[api];
  });
}

export { registerApiGlobally };
