var _setup = require('./_setup.js');
var _unmethodize = require('./_unmethodize.js');

var push = _unmethodize(_setup.ArrayProto.push);

module.exports = push;
