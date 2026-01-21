import { twMerge } from "tailwind-merge";

type HeadingProps = { level?: 1 | 2 | 3 | 4 | 5 | 6 } & React.ComponentPropsWithoutRef<
  "h1" | "h2" | "h3" | "h4" | "h5" | "h6"
>;

export function Heading({ className, level = 1, ...props }: HeadingProps) {
  const Element: `h${typeof level}` = `h${level}`;

  return (
    <Element
      {...props}
      className={twMerge("text-xl/8 font-medium text-gray-800 sm:text-xl/8 dark:text-white", className)}
    />
  );
}

export function Subheading({ className, level = 2, ...props }: HeadingProps) {
  const Element: `h${typeof level}` = `h${level}`;

  return (
    <Element
      {...props}
      className={twMerge("text-base/7 font-semibold text-gray-800sm:text-sm/6 dark:text-white", className)}
    />
  );
}
