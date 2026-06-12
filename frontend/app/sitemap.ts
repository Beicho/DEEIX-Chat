import type { MetadataRoute } from "next";

const DEFAULT_SITE_URL = "https://chat.windhub.cc";

export const dynamic = "force-static";

function siteURL(): string {
  const configured = process.env.NEXT_PUBLIC_SITE_URL || process.env.PUBLIC_WEB_BASE_URL || DEFAULT_SITE_URL;
  return configured.trim().replace(/\/+$/, "") || DEFAULT_SITE_URL;
}

export default function sitemap(): MetadataRoute.Sitemap {
  const baseURL = siteURL();
  const lastModified = new Date();
  return [
    {
      url: baseURL,
      lastModified,
      changeFrequency: "daily",
      priority: 1,
    },
    {
      url: `${baseURL}/status`,
      lastModified,
      changeFrequency: "hourly",
      priority: 0.7,
    },
    {
      url: `${baseURL}/login`,
      lastModified,
      changeFrequency: "monthly",
      priority: 0.5,
    },
  ];
}
