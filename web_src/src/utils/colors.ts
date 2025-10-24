export const getColorClass = (color?: string): string => {
  switch (color) {
    case 'blue':
      return 'text-blue-600 dark:text-blue-400'
    case 'green':
      return 'text-green-600 dark:text-green-400'
    case 'red':
      return 'text-red-600 dark:text-red-400'
    case 'yellow':
      return 'text-yellow-600 dark:text-yellow-400'
    case 'purple':
      return 'text-purple-600 dark:text-purple-400'
    case 'orange':
      return 'text-orange-600 dark:text-orange-400'
    case 'pink':
      return 'text-pink-600 dark:text-pink-400'
    case 'indigo':
      return 'text-indigo-600 dark:text-indigo-400'
    default:
      return 'text-blue-600 dark:text-blue-400'
  }
}
