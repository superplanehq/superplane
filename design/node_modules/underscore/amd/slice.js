define(['./_unmethodize', './_setup'], function (_unmethodize, _setup) {

	var slice = _unmethodize(_setup.ArrayProto.slice);

	return slice;

});
