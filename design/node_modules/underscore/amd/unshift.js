define(['./_unmethodize', './_setup'], function (_unmethodize, _setup) {

	var unshift = _unmethodize(_setup.ArrayProto.unshift);

	return unshift;

});
