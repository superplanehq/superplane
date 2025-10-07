/* eslint-disable @typescript-eslint/no-explicit-any */

/**
 * Flattens a JSON object by parent field and depth layer for autocomplete
 * @param {Object} obj - The JSON object to flatten
 * @returns {Object} Flattened structure with parent-depth as keys and field arrays as values
 */
export function flattenForAutocomplete(obj: any) {
    const result: any = {};
    
    // Add root level keys
    if (typeof obj === 'object' && !Array.isArray(obj)) {
      result['root-0'] = Object.keys(obj);
    }
    
    function traverse(current: any, parentKey: string | null = null, depth = 0) {
      if (current === null || current === undefined) {
        return;
      }
      
      // Handle arrays
      if (Array.isArray(current)) {
        current.forEach((item, index) => {
          const arrayKey = `${parentKey}[${index}]`;
          
          // Add the array index accessor to parent's suggestions
          if (parentKey) {
            const parentDepthKey = `${parentKey}-${depth}`;
            if (!result[parentDepthKey]) {
              result[parentDepthKey] = [];
            }
            if (!result[parentDepthKey].includes(arrayKey)) {
              result[parentDepthKey].push(arrayKey);
            }
          }
          
          // Traverse into array item
          traverse(item, arrayKey, depth + 1);
        });
        return;
      }
      
      // Handle objects
      if (typeof current === 'object') {
        const keys = Object.keys(current);
        
        if (parentKey !== null) {
          const depthKey = `${parentKey}-${depth}`;
          if (!result[depthKey]) {
            result[depthKey] = [];
          }
          // Add all direct child keys
          keys.forEach(key => {
            if (!result[depthKey].includes(key)) {
              result[depthKey].push(key);
            }
          });
        }
        
        // Traverse into each property
        keys.forEach(key => {
          const newParent = parentKey ? `${parentKey}.${key}` : key;
          traverse(current[key], newParent, parentKey === null ? 0 : depth + 1);
        });
        return;
      }
      
      // Primitive values (string, number, boolean) - no further traversal needed
    }
    
    traverse(obj);
    return result;
  }
  
  /**
   * Get autocomplete suggestions based on current input path
   * @param {Object} flattenedData - The flattened data structure
   * @param {string} currentPath - The current input path (e.g., "test.myArray[0]")
   * @returns {Array} Array of suggestion strings
   */
  export function getAutocompleteSuggestions(flattenedData: any, currentPath: string) {
    if (!currentPath) {
      const topLevelKeys = Object.keys(flattenedData).filter(key => key.endsWith('-0'));


      if (topLevelKeys.length > 0) {
        return topLevelKeys.map(topLevelKey => {
            const splittedTopLevelKey = topLevelKey.split('-');
            const keyWords = splittedTopLevelKey.slice(0, splittedTopLevelKey.length - 1);
            return keyWords.join('-');
        })
      }

      return [];
    }
    
    const depth = (currentPath.match(/\./g) || []).length + 
                  (currentPath.match(/\[/g) || []).length;
    
    const lookupKey = `${currentPath}-${depth}`;
    return flattenedData[lookupKey] || [];
  }