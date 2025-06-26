'use strict';

var define = require('define-properties');
var RequireObjectCoercible = require('es-abstract/2024/RequireObjectCoercible');
var callBind = require('call-bind');
var callBound = require('call-bound');

var implementation = require('./implementation');

var getPolyfill = require('./polyfill');
var polyfill = callBind.apply(getPolyfill());

var shim = require('./shim');

var $slice = callBound('%Array.prototype.slice%');

// eslint-disable-next-line no-unused-vars
var boundShim = function reduce(array, callbackfn) {
	RequireObjectCoercible(array);
	return polyfill(array, $slice(arguments, 1));
};
define(boundShim, {
	getPolyfill: getPolyfill,
	implementation: implementation,
	shim: shim
});

module.exports = boundShim;
