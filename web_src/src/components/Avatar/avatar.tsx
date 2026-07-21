import * as Headless from "@headlessui/react";
import clsx from "clsx";
import React, { forwardRef, useEffect, useState } from "react";
import { Link } from "../Link/link";

type AvatarProps = {
  src?: string | null;
  square?: boolean;
  initials?: string;
  alt?: string;
  className?: string;
};

export function Avatar({
  src = null,
  square = false,
  initials,
  alt = "",
  className,
  ...props
}: AvatarProps & React.ComponentPropsWithoutRef<"span">) {
  // Track image load failures (e.g. a 404 avatar URL for a user without a
  // photo, or a bot account with no GitHub avatar) so we can fall back to the
  // initials / generic placeholder instead of leaving the browser's
  // broken-image icon on screen.
  const [imageFailed, setImageFailed] = useState(false);

  // Reset the failure flag whenever the source changes so a fresh (possibly
  // valid) URL gets another chance to load after a previous one failed.
  useEffect(() => {
    setImageFailed(false);
  }, [src]);

  const showImage = Boolean(src) && !imageFailed;
  const showInitials = Boolean(initials) && !showImage;
  const showPlaceholder = !showImage && !showInitials;

  return (
    <span
      data-slot="avatar"
      {...props}
      className={clsx(
        className,
        // Basic layout
        "inline-grid shrink-0 align-middle [--avatar-radius:20%] *:col-start-1 *:row-start-1",
        "outline -outline-offset-1 outline-black/10 dark:outline-white/10",
        // Border radius
        square ? "rounded-(--avatar-radius) *:rounded-(--avatar-radius)" : "rounded-full *:rounded-full",
      )}
    >
      {showInitials && (
        <svg
          className="size-full fill-current p-[5%] text-[48px] font-medium uppercase select-none"
          viewBox="0 0 100 100"
          aria-hidden={alt ? undefined : "true"}
        >
          {alt && <title>{alt}</title>}
          <text x="50%" y="50%" alignmentBaseline="middle" dominantBaseline="middle" textAnchor="middle" dy=".125em">
            {initials}
          </text>
        </svg>
      )}
      {showPlaceholder && <PlaceholderIcon alt={alt} />}
      {showImage && <img className="size-full" src={src!} alt={alt} onError={() => setImageFailed(true)} />}
    </span>
  );
}

/**
 * Generic user silhouette shown when there is no image to display and no
 * initials to fall back to (either the avatar had neither, or the image URL
 * failed to load and no initials were provided).
 */
function PlaceholderIcon({ alt }: { alt?: string }) {
  return (
    <svg
      className="size-full fill-current p-[15%] opacity-60 select-none"
      viewBox="0 0 24 24"
      aria-hidden={alt ? undefined : "true"}
    >
      {alt && <title>{alt}</title>}
      <path d="M12 12a5 5 0 1 0 0-10 5 5 0 0 0 0 10Zm0 2c-4.42 0-8 2.24-8 5v1h16v-1c0-2.76-3.58-5-8-5Z" />
    </svg>
  );
}

export const AvatarButton = forwardRef(function AvatarButton(
  {
    src,
    square = false,
    initials,
    alt,
    className,
    ...props
  }: AvatarProps &
    (Omit<Headless.ButtonProps, "as" | "className"> | Omit<React.ComponentPropsWithoutRef<typeof Link>, "className">),
  ref: React.ForwardedRef<HTMLElement>,
) {
  const classes = clsx(
    className,
    square ? "rounded-[20%]" : "rounded-full",
    "relative inline-grid focus:not-data-focus:outline-hidden data-focus:outline-2 data-focus:outline-offset-2 data-focus:outline-blue-500",
  );

  return "href" in props ? (
    <Link {...props} className={classes} ref={ref as React.ForwardedRef<HTMLAnchorElement>}>
      <TouchTarget>
        <Avatar src={src} square={square} initials={initials} alt={alt} />
      </TouchTarget>
    </Link>
  ) : (
    <Headless.Button {...props} className={classes} ref={ref}>
      <TouchTarget>
        <Avatar src={src} square={square} initials={initials} alt={alt} />
      </TouchTarget>
    </Headless.Button>
  );
});

function TouchTarget({ children }: { children: React.ReactNode }) {
  return (
    <>
      <span
        className="absolute top-1/2 left-1/2 size-[max(100%,2.75rem)] -translate-x-1/2 -translate-y-1/2 pointer-fine:hidden"
        aria-hidden="true"
      />
      {children}
    </>
  );
}
