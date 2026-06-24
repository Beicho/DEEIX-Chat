import type { Metadata } from "next";
import { Suspense } from "react";

import { PublicSharePage } from "@/features/share/components/public-share-page";

export function generateMetadata(): Metadata {
  return {
    title: "DEEIX Chat",
    description: "DEEIX Chat is a multi-model AI conversation workspace.",
    openGraph: {
      title: "DEEIX Chat",
      description: "DEEIX Chat is a multi-model AI conversation workspace.",
      type: "website",
    },
    twitter: {
      card: "summary",
      title: "DEEIX Chat",
      description: "DEEIX Chat is a multi-model AI conversation workspace.",
    },
  };
}

export default function Page() {
  return (
    <Suspense fallback={null}>
      <PublicSharePage />
    </Suspense>
  );
}
