'use strict';

var CreateDataProperty = require('es-abstract/2024/CreateDataProperty');
var RequireObjectCoercible = require('es-object-atoms/RequireObjectCoercible');
var ToObject = require('es-object-atoms/ToObject');
var safeConcat = require('safe-array-concat');
var reduce = require('array.prototype.reduce');
var gOPD = require('gopd');
var $Object = require('es-object-atoms');

var $getOwnNames = $Object.getOwnPropertyNames;
var $getSymbols = $Object.getOwnPropertySymbols;

var getAll = $getSymbols ? function (obj) {
	return safeConcat($getOwnNames(obj), $getSymbols(obj));
} : $getOwnNames;

var isES5 = gOPD && typeof $getOwnNames === 'function';

module.exports = function getOwnPropertyDescriptors(value) {
	RequireObjectCoercible(value);
	if (!isES5) {
		throw new TypeError('getOwnPropertyDescriptors requires Object.getOwnPropertyDescriptor');
	}

	var O = ToObject(value);
	return reduce(
		getAll(O),
		function (acc, key) {
			var descriptor = gOPD(O, key);
			if (typeof descriptor !== 'undefined') {
				CreateDataProperty(acc, key, descriptor);
			}
			return acc;
		},
		{}
	);
};
