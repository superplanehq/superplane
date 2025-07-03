import fs from 'fs';
import path from 'path';
import appRoot from 'app-root-dir';
import dotenv from 'dotenv';
import dotenvExpand from 'dotenv-expand';

const dotEnvBase = path.join(appRoot.get(), ".env");
function getEnvironment({ nodeEnv, buildTarget } = {}) {
  let raw = {};
  const stringified = {};
  const webpack = { "process.env": stringified };
  const NODE_ENV = typeof nodeEnv === "undefined" ? process.env.NODE_ENV : nodeEnv;
  const BUILD_TARGET = typeof nodeEnv === "undefined" ? process.env.BUILD_TARGET : buildTarget;
  const dotenvFiles = [
    BUILD_TARGET && NODE_ENV && `${dotEnvBase}.${BUILD_TARGET}.${NODE_ENV}.local`,
    BUILD_TARGET && NODE_ENV && `${dotEnvBase}.${BUILD_TARGET}.${NODE_ENV}`,
    BUILD_TARGET && NODE_ENV !== "test" && `${dotEnvBase}.${BUILD_TARGET}.local`,
    BUILD_TARGET && `${dotEnvBase}.${BUILD_TARGET}`,
    NODE_ENV && `${dotEnvBase}.${NODE_ENV}.local`,
    NODE_ENV && `${dotEnvBase}.${NODE_ENV}`,
    NODE_ENV !== "test" && `${dotEnvBase}.local`,
    dotEnvBase
  ].filter(Boolean);
  dotenvFiles.forEach((dotenvFile) => {
    if (fs.existsSync(dotenvFile)) {
      const config = dotenv.config({
        path: dotenvFile
      });
      raw = Object.assign({}, raw, dotenvExpand.expand(config).parsed);
    }
  });
  Object.keys(raw).forEach((key) => {
    stringified[key] = JSON.stringify(raw[key]);
  });
  return { raw, stringified, webpack };
}

export { getEnvironment };
