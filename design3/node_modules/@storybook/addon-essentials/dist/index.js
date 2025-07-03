'use strict';

var path = require('path');
var nodeLogger = require('@storybook/node-logger');
var coreCommon = require('@storybook/core-common');

function _interopDefault (e) { return e && e.__esModule ? e : { default: e }; }

var path__default = /*#__PURE__*/_interopDefault(path);

var requireMain=configDir=>{let absoluteConfigDir=path__default.default.isAbsolute(configDir)?configDir:path__default.default.join(process.cwd(),configDir),mainFile=path__default.default.join(absoluteConfigDir,"main");return coreCommon.serverRequire(mainFile)??{}};function addons(options){let checkInstalled=(addonName,main2)=>{var _a;let addon=`@storybook/addon-${addonName}`,existingAddon=(_a=main2.addons)==null?void 0:_a.find(entry=>{let name=typeof entry=="string"?entry:entry.name;return name==null?void 0:name.startsWith(addon)});return existingAddon&&nodeLogger.logger.info(`Found existing addon ${JSON.stringify(existingAddon)}, skipping.`),!!existingAddon},main=requireMain(options.configDir);return ["docs","controls","actions","backgrounds","viewport","toolbars","measure","outline","highlight"].filter(key=>options[key]!==!1).filter(addon=>!checkInstalled(addon,main)).map(addon=>`@storybook/addon-essentials/${addon}`)}

exports.addons = addons;
