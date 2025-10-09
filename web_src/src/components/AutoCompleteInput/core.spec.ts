import { describe, it, expect, beforeEach } from 'vitest';
import { flattenForAutocomplete, getAutocompleteSuggestions } from './core'

/* eslint-disable @typescript-eslint/no-explicit-any */


describe('flattenForAutocomplete', () => {
  it('should flatten a simple object with root level keys', () => {
    const obj = { name: 'John', age: 30 };
    const result = flattenForAutocomplete(obj);
    
    expect(result['root-0']).toEqual(['name', 'age']);
  });

  it('should flatten nested objects', () => {
    const obj = {
      user: {
        name: 'John',
        address: {
          city: 'NYC'
        }
      }
    };
    const result = flattenForAutocomplete(obj);
    
    expect(result['root-0']).toEqual(['user']);
    expect(result['user-0']).toEqual(['name', 'address']);
    expect(result['user.address-1']).toEqual(['city']);
  });

  it('should handle arrays correctly', () => {
    const obj = {
      items: [
        { id: 1, name: 'Item 1' },
        { id: 2, name: 'Item 2' }
      ]
    };
    const result = flattenForAutocomplete(obj);
    
    expect(result['root-0']).toEqual(['items']);
    expect(result['items-0']).toContain('items[0]');
    expect(result['items-0']).toContain('items[1]');
    expect(result['items[0]-1']).toEqual(['id', 'name']);
    expect(result['items[1]-1']).toEqual(['id', 'name']);
  });

  it('should handle nested arrays and objects', () => {
    const obj = {
      someField: {
        adasd: [
          {
            test: {
              dasdsa: 'dsadsad',
              teasdas: 123213
            }
          }
        ]
      }
    };
    const result = flattenForAutocomplete(obj);
    
    expect(result['root-0']).toEqual(['someField']);
    expect(result['someField-0']).toEqual(['adasd']);
    expect(result['someField.adasd-1']).toContain('someField.adasd[0]');
    expect(result['someField.adasd[0]-2']).toEqual(['test']);
    expect(result['someField.adasd[0].test-3']).toEqual(['dasdsa', 'teasdas']);
  });

  it('should handle null and undefined values', () => {
    const obj = {
      value1: null,
      value2: undefined,
      nested: {
        value3: null
      }
    };
    const result = flattenForAutocomplete(obj);
    
    expect(result['root-0']).toEqual(['value1', 'value2', 'nested']);
    expect(result['nested-0']).toEqual(['value3']);
  });

  it('should handle empty objects', () => {
    const obj = {};
    const result = flattenForAutocomplete(obj);
    
    expect(result['root-0']).toEqual([]);
  });

  it('should handle empty arrays', () => {
    const obj = {
      items: []
    };
    const result = flattenForAutocomplete(obj);
    
    expect(result['root-0']).toEqual(['items']);
    expect(result['items-0']).toBeUndefined();
  });
});

describe('getAutocompleteSuggestions', () => {
  const testData = {
    someField: {
      adasd: [
        {
          test: {
            dasdsa: 'dsadsad',
            teasdas: 123213
          }
        }
      ]
    }
  };
  
  let flattened: any;

  beforeEach(() => {
    flattened = flattenForAutocomplete(testData);
  });

  it('should return root level keys when path is empty', () => {
    const suggestions = getAutocompleteSuggestions(flattened, '');
    
    expect(suggestions).toContain('someField');
  });

  it('should return nested keys for a given path', () => {
    const suggestions = getAutocompleteSuggestions(flattened, 'someField');
    
    expect(suggestions).toEqual(['adasd']);
  });

  it('should return array indices for array paths', () => {
    const suggestions = getAutocompleteSuggestions(flattened, 'someField.adasd');
    
    expect(suggestions).toContain('someField.adasd[0]');
  });

  it('should return object keys after array index', () => {
    const suggestions = getAutocompleteSuggestions(flattened, 'someField.adasd[0]');
    
    expect(suggestions).toEqual(['test']);
  });

  it('should return nested object keys', () => {
    const suggestions = getAutocompleteSuggestions(flattened, 'someField.adasd[0].test');
    
    expect(suggestions).toEqual(expect.arrayContaining(['dasdsa', 'teasdas']));
    expect(suggestions).toHaveLength(2);
  });

  it('should return empty array for non-existent paths', () => {
    const suggestions = getAutocompleteSuggestions(flattened, 'nonexistent.path');
    
    expect(suggestions).toEqual([]);
  });

  it('should handle deep nesting correctly', () => {
    const deepData = {
      level1: {
        level2: {
          level3: {
            level4: 'value'
          }
        }
      }
    };
    const deepFlattened = flattenForAutocomplete(deepData);
    
    expect(getAutocompleteSuggestions(deepFlattened, '')).toContain('level1');
    expect(getAutocompleteSuggestions(deepFlattened, 'level1')).toEqual(['level2']);
    expect(getAutocompleteSuggestions(deepFlattened, 'level1.level2')).toEqual(['level3']);
    expect(getAutocompleteSuggestions(deepFlattened, 'level1.level2.level3')).toEqual(['level4']);
  });

  it('should handle multiple array items', () => {
    const arrayData = {
      users: [
        { name: 'Alice', age: 30 },
        { name: 'Bob', age: 25 }
      ]
    };
    const arrayFlattened = flattenForAutocomplete(arrayData);
    
    const suggestions = getAutocompleteSuggestions(arrayFlattened, 'users');
    expect(suggestions).toContain('users[0]');
    expect(suggestions).toContain('users[1]');
    
    const user0Suggestions = getAutocompleteSuggestions(arrayFlattened, 'users[0]');
    expect(user0Suggestions).toEqual(expect.arrayContaining(['name', 'age']));
  });
});

describe('Integration tests', () => {
  it('should provide correct autocomplete flow for complex nested structure', () => {
    const data = {
      company: {
        departments: [
          {
            name: 'Engineering',
            employees: [
              { id: 1, name: 'Alice' },
              { id: 2, name: 'Bob' }
            ]
          }
        ]
      }
    };
    
    const flattened = flattenForAutocomplete(data);
    
    // Start typing
    expect(getAutocompleteSuggestions(flattened, '')).toContain('company');
    
    // Type "company."
    expect(getAutocompleteSuggestions(flattened, 'company')).toEqual(['departments']);
    
    // Type "company.departments"
    expect(getAutocompleteSuggestions(flattened, 'company.departments')).toContain('company.departments[0]');
    
    // Type "company.departments[0]."
    expect(getAutocompleteSuggestions(flattened, 'company.departments[0]')).toEqual(expect.arrayContaining(['name', 'employees']));
    
    // Type "company.departments[0].employees"
    expect(getAutocompleteSuggestions(flattened, 'company.departments[0].employees')).toContain('company.departments[0].employees[0]');
    
    // Type "company.departments[0].employees[0]."
    expect(getAutocompleteSuggestions(flattened, 'company.departments[0].employees[0]')).toEqual(expect.arrayContaining(['id', 'name']));
  });
});