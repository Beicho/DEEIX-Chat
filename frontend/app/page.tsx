import type { Metadata } from "next";

import enLanding from "@/i18n/messages/en-US/landing.json";
import zhLanding from "@/i18n/messages/zh-CN/landing.json";
import { PublicLandingPage } from "@/features/landing/components/public-landing-page";
import { normalizeAppLocale } from "@/i18n/config";

export const dynamic = "force-static";

function landingMetadata() {
  const locale = normalizeAppLocale(process.env.NEXT_PUBLIC_BUILD_LOCALE);
  return locale === "en-US" ? enLanding.metadata : zhLanding.metadata;
}

export function generateMetadata(): Metadata {
  const metadata = landingMetadata();
  return {
    title: metadata.title,
    description: metadata.description,
    openGraph: {
      title: metadata.title,
      description: metadata.description,
      type: "website",
    },
    twitter: {
      card: "summary_large_image",
      title: metadata.title,
      description: metadata.description,
    },
  };
}

export default function Page() {
  return <PublicLandingPage />;
}
