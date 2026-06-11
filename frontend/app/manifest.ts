import type { MetadataRoute } from "next";

import { normalizeAppLocale } from "@/i18n/config";
import { pwaAsset } from "@/shared/pwa/assets";

export const dynamic = "force-static";

export default function manifest(): MetadataRoute.Manifest {
  const locale = normalizeAppLocale(process.env.NEXT_PUBLIC_BUILD_LOCALE);
  const isChinese = locale === "zh-CN";

  return {
    name: isChinese ? "DEEIX Chat 多模型对话" : "DEEIX Chat",
    short_name: "DEEIX",
    description: isChinese
      ? "DEEIX Chat 是一个多模型 AI 对话工作台。"
      : "DEEIX Chat is a multi-model AI conversation workspace.",
    id: "/",
    start_url: "/chat",
    scope: "/",
    display: "standalone",
    orientation: "any",
    categories: ["productivity", "business", "utilities"],
    lang: locale,
    icons: [
      {
        src: pwaAsset("/pwa/icon-192.png"),
        sizes: "192x192",
        type: "image/png",
        purpose: "any",
      },
      {
        src: pwaAsset("/pwa/icon-512.png"),
        sizes: "512x512",
        type: "image/png",
        purpose: "any",
      },
      {
        src: pwaAsset("/pwa/icon-maskable-512.png"),
        sizes: "512x512",
        type: "image/png",
        purpose: "maskable",
      },
    ],
  };
}
