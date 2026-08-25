"use client";

import { useBranding } from "@/shared/components/branding-provider";
import { useTheme } from "@/shared/components/theme-provider";

type AppLogoProps = {
  alt?: string;
  width: number;
  height: number;
  priority?: boolean;
  className?: string;
};

export function AppLogo({
  alt,
  width,
  height,
  priority,
  className,
}: AppLogoProps) {
  const { resolvedTheme } = useTheme();
  const branding = useBranding();
  const src = resolvedTheme === "dark"
    ? branding.logoDarkURL || branding.logoURL || "/logo-white.svg"
    : branding.logoURL || "/logo.svg";

  return (
    // eslint-disable-next-line @next/next/no-img-element -- Branding logos can be arbitrary admin-configured URLs.
    <img
      src={src}
      alt={alt || branding.appName}
      width={width}
      height={height}
      loading={priority ? "eager" : "lazy"}
      className={className}
    />
  );
}
