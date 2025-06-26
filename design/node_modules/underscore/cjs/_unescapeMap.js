var _escapeMap = require('./_escapeMap.js');
var invert = require('./invert.js');

// Internal list of HTML entities for unescaping.
var unescapeMap = invert(_escapeMap);

module.exports = unescapeMap;
