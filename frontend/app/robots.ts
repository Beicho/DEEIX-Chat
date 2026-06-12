import type { MetadataRoute } from "next";

const DEFAULT_SITE_URL = "https://chat.windhub.cc";

export const dynamic = "force-static";

function siteURL(): string {
  const configured = process.env.NEXT_PUBLIC_SITE_URL || process.env.PUBLIC_WEB_BASE_URL || DEFAULT_SITE_URL;
  return configured.trim().replace(/\/+$/, "") || DEFAULT_SITE_URL;
}

export default function robots(): MetadataRoute.Robots {
  return {
    rules: {
      userAgent: "*",
      allow: "/",
      disallow: ["/admin", "/api"],
    },
    sitemap: `${siteURL()}/sitemap.xml`,
  };
}
