define(['./_unmethodize', './_setup'], function (_unmethodize, _setup) {

	var concat = _unmethodize(_setup.ArrayProto.concat);

	return concat;

});
