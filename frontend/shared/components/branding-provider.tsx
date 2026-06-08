"use client";

import * as React from "react";

import { getBrandingSettings, type BrandingSettings } from "@/shared/api/settings";

const DEFAULT_BRANDING: BrandingSettings = {
  appName: "DEEIX Chat",
  logoURL: "/logo.svg",
  logoDarkURL: "/logo-white.svg",
};

const BrandingContext = React.createContext<BrandingSettings>(DEFAULT_BRANDING);

export function BrandingProvider({ children }: { children: React.ReactNode }) {
  const [branding, setBranding] = React.useState<BrandingSettings>(DEFAULT_BRANDING);
  const appliedTitleRef = React.useRef(DEFAULT_BRANDING.appName);

  const loadBranding = React.useCallback(async () => {
    try {
      const data = await getBrandingSettings();
      setBranding({
        appName: data.appName?.trim() || DEFAULT_BRANDING.appName,
        logoURL: data.logoURL?.trim() || DEFAULT_BRANDING.logoURL,
        logoDarkURL: data.logoDarkURL?.trim() || DEFAULT_BRANDING.logoDarkURL,
      });
    } catch {
      setBranding(DEFAULT_BRANDING);
    }
  }, []);

  React.useEffect(() => {
    void loadBranding();
    window.addEventListener("deeix:branding-updated", loadBranding);
    return () => window.removeEventListener("deeix:branding-updated", loadBranding);
  }, [loadBranding]);

  React.useEffect(() => {
    const currentTitle = document.title.trim();
    const previousTitle = appliedTitleRef.current;
    if (!currentTitle || currentTitle === DEFAULT_BRANDING.appName || currentTitle === previousTitle) {
      document.title = branding.appName;
      appliedTitleRef.current = branding.appName;
    }
  }, [branding.appName]);

  return <BrandingContext.Provider value={branding}>{children}</BrandingContext.Provider>;
}

export function useBranding() {
  return React.useContext(BrandingContext);
}
