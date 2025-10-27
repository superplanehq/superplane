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

export const getBackgroundColorClass = (color?: string): string => {
  switch (color) {
    case 'blue':
      return 'bg-blue-100'
    case 'green':
      return 'bg-green-100'
    case 'red':
      return 'bg-red-100'
    case 'yellow':
      return 'bg-yellow-100'
    case 'purple':
      return 'bg-purple-100'
    case 'orange':
      return 'bg-orange-100'
    case 'pink':
      return 'bg-pink-100'
    case 'indigo':
      return 'bg-indigo-100'
    case 'sky':
      return 'bg-sky-100'
    case 'gray':
      return 'bg-gray-100'
    default:
      return 'bg-gray-100'
  }
}
