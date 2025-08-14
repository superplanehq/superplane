import clsx from 'clsx'

export interface MaterialSymbolProps {
  /** The name of the Material Symbol (e.g., 'home', 'settings', 'person') */
  name: string
  /** Size variant - supports both named sizes and pixel sizes */
  size?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl' | '4xl' | '5xl' | '6xl' | '7xl' | number
  /** Fill variant (0 = outlined, 1 = filled) */
  fill?: 0 | 1
  /** Weight variant (100-700) */
  weight?: 100 | 200 | 300 | 400 | 500 | 600 | 700
  /** Grade variant (-25 to 200) */
  grade?: number
  /** Optical size (20-48) */
  opticalSize?: number
  /** Additional CSS classes */
  className?: string
  /** Data slot attribute for button styling */
  'data-slot'?: string
}

export function MaterialSymbol({
  name,
  size = 'md',
  fill = 0,
  weight = 400,
  grade = 0,
  opticalSize = 24,
  className,
  'data-slot': dataSlot
}: MaterialSymbolProps) {
  // Named size classes
  const namedSizeClasses = {
    sm: '!text-sm', // 14px
    md: '!text-base', // 16px
    lg: '!text-xl', // 20px
    xl: '!text-2xl', // 24px
    '2xl': '!text-2xl', // 24px
    '3xl': '!text-3xl', // 30px
    '4xl': '!text-4xl', // 36px
    '5xl': '!text-5xl', // 48px
    '6xl': '!text-6xl', // 60px
    '7xl': '!text-7xl' // 72px
  }

  // Pixel size classes
  const pixelSizeClasses = {
    32: '!w-8 !h-8 !text-[32px] !leading-8',
    36: '!w-9 !h-9 !text-[36px] !leading-9',
    40: '!w-10 !h-10 !text-[40px] !leading-10',
    48: '!w-12 !h-12 !text-[48px] !leading-12',
    56: '!w-14 !h-14 !text-[56px] !leading-14',
    60: '!w-[60px] !h-[60px] !text-[60px] !leading-[60px]',
    64: '!w-16 !h-16 !text-[64px] !leading-16'
  }

  // Determine which size class to use
  const getSizeClass = () => {
    if (typeof size === 'number') {
      // Check if we have a predefined class for this size
      if (pixelSizeClasses[size as keyof typeof pixelSizeClasses]) {
        return pixelSizeClasses[size as keyof typeof pixelSizeClasses]
      }
      // Otherwise, generate custom classes for any numeric size
      return `!w-[${size}px] !h-[${size}px] !text-[${size}px] !leading-[${size}px]`
    }
    return namedSizeClasses[size as keyof typeof namedSizeClasses] || namedSizeClasses.md
  }

  const style = {
    fontVariationSettings: `'FILL' ${fill}, 'wght' ${weight}, 'GRAD' ${grade}, 'opsz' ${opticalSize}`
  }

  return (
    <span 
      className={clsx(
        'material-symbols-outlined select-none inline-flex items-center justify-center',
        getSizeClass(),
        className
      )}
      style={style}
      aria-hidden="true"
      data-slot={dataSlot}
    >
      {name}
    </span>
  )
}

// Preset variants for common use cases
export const MaterialSymbolFilled = (props: Omit<MaterialSymbolProps, 'fill'>) => (
  <MaterialSymbol {...props} fill={1} />
)

export const MaterialSymbolLight = (props: Omit<MaterialSymbolProps, 'weight'>) => (
  <MaterialSymbol {...props} weight={300} />
)

export const MaterialSymbolBold = (props: Omit<MaterialSymbolProps, 'weight'>) => (
  <MaterialSymbol {...props} weight={600} />
)