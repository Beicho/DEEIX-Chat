"use client";

import * as React from "react";
import Link from "next/link";
import { ArrowRight, Check, Layers3, MessageSquareText, Share2, Smartphone, Sparkles } from "lucide-react";
import { useLocale, useTranslations } from "next-intl";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { listPublicBillingPlans } from "@/shared/api/billing";
import type { BillingPlanDTO, BillingPlanPriceDTO } from "@/shared/api/billing.types";
import { listPublicModelCatalog } from "@/shared/api/model";
import type { PublicModelDTO } from "@/shared/api/model.types";
import { AppLogo } from "@/shared/components/app-logo";
import { cn } from "@/lib/utils";

const FEATURE_ICONS = {
  models: Layers3,
  projects: Sparkles,
  sharing: Share2,
  mobile: Smartphone,
};

const HERO_PREVIEW_KEYS = ["models", "projects", "sharing"] as const;

function formatUSD(value: number, locale: string, options: Intl.NumberFormatOptions = {}): string {
  return new Intl.NumberFormat(locale, {
    style: "currency",
    currency: "USD",
    maximumFractionDigits: 4,
    ...options,
  }).format(Number.isFinite(value) ? value : 0);
}

function defaultPrice(plan: BillingPlanDTO): BillingPlanPriceDTO | null {
  return plan.prices.find((price) => price.isDefault) ?? plan.prices[0] ?? null;
}

function modelStartPrice(model: PublicModelDTO): number | null {
  const pricing = model.pricing;
  if (!pricing) return null;
  if (pricing.isFree) return 0;
  if (pricing.mode === "call") return pricing.callUSDPerCall;
  if (pricing.mode === "duration") return pricing.durationUSDPerSecond;
  if (pricing.mode === "tiered" && pricing.tiers.length > 0) {
    return Math.min(...pricing.tiers.map((tier) => tier.inputUSDPerMTokens).filter((value) => value > 0));
  }
  const candidates = [
    pricing.inputUSDPerMTokens,
    pricing.outputUSDPerMTokens,
    pricing.cacheReadUSDPerMTokens,
  ].filter((value) => value > 0);
  return candidates.length ? Math.min(...candidates) : null;
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-xl border border-border bg-card/70 px-3 py-3 text-card-foreground shadow-xs">
      <div className="text-xl font-semibold tracking-tight">{value}</div>
      <div className="mt-1 text-xs text-muted-foreground">{label}</div>
    </div>
  );
}

function FeatureCard({ featureKey }: { featureKey: keyof typeof FEATURE_ICONS }) {
  const t = useTranslations("landing.features.items");
  const Icon = FEATURE_ICONS[featureKey];
  return (
    <Card className="gap-0 py-0 shadow-xs">
      <CardContent className="flex min-h-36 flex-col gap-3 p-4">
        <span className="flex size-8 items-center justify-center rounded-lg bg-primary/10 text-primary">
          <Icon className="size-4" strokeWidth={1.7} />
        </span>
        <div className="space-y-1">
          <h3 className="text-sm font-semibold text-foreground">{t(`${featureKey}.title`)}</h3>
          <p className="text-xs leading-5 text-muted-foreground">{t(`${featureKey}.description`)}</p>
        </div>
      </CardContent>
    </Card>
  );
}

function ModelCatalog({ models, loading }: { models: PublicModelDTO[]; loading: boolean }) {
  const t = useTranslations("landing.models");
  const locale = useLocale();
  const visibleModels = models.slice(0, 8);

  return (
    <section className="space-y-4">
      <div className="flex flex-col gap-1 md:flex-row md:items-end md:justify-between">
        <div>
          <h2 className="text-xl font-semibold tracking-tight md:text-2xl">{t("title")}</h2>
          <p className="mt-1 max-w-2xl text-sm text-muted-foreground">{t("description")}</p>
        </div>
      </div>
      <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
        {loading ? Array.from({ length: 8 }).map((_, index) => (
          <div key={index} className="rounded-xl border border-border bg-card p-3">
            <Skeleton className="h-4 w-3/4" />
            <Skeleton className="mt-3 h-3 w-1/2" />
          </div>
        )) : visibleModels.length ? visibleModels.map((model) => {
          const price = modelStartPrice(model);
          return (
            <div key={model.platformModelName} className="rounded-xl border border-border bg-card/70 p-3 shadow-xs">
              <div className="truncate text-sm font-medium text-foreground">{model.platformModelName}</div>
              <div className="mt-1 flex min-w-0 items-center justify-between gap-2 text-xs text-muted-foreground">
                <span className="truncate">{model.vendor || model.icon || model.platformModelName}</span>
                <Badge variant="secondary">
                  {price === 0 ? t("free") : price ? t("from", { price: formatUSD(price, locale) }) : t("empty")}
                </Badge>
              </div>
            </div>
          );
        }) : (
          <div className="rounded-xl border border-border bg-card p-4 text-sm text-muted-foreground sm:col-span-2 lg:col-span-4">
            {t("empty")}
          </div>
        )}
      </div>
    </section>
  );
}

function PlansPreview({ plans, loading }: { plans: BillingPlanDTO[]; loading: boolean }) {
  const t = useTranslations("landing.plans");
  const locale = useLocale();
  const visiblePlans = plans.filter((plan) => plan.isActive).slice(0, 3);

  return (
    <section className="space-y-4">
      <div>
        <h2 className="text-xl font-semibold tracking-tight md:text-2xl">{t("title")}</h2>
        <p className="mt-1 max-w-2xl text-sm text-muted-foreground">{t("description")}</p>
      </div>
      <div className="grid gap-3 md:grid-cols-3">
        {loading ? Array.from({ length: 3 }).map((_, index) => (
          <Card key={index} className="py-0 shadow-xs">
            <CardContent className="p-4">
              <Skeleton className="h-5 w-24" />
              <Skeleton className="mt-4 h-4 w-32" />
              <Skeleton className="mt-3 h-12 w-full" />
            </CardContent>
          </Card>
        )) : visiblePlans.length ? visiblePlans.map((plan) => {
          const price = defaultPrice(plan);
          const amount = price ? formatUSD(price.amountCents / 100, locale, { maximumFractionDigits: 2 }) : t("empty");
          const interval = price ? t(`intervals.${price.billingInterval}`) : "";
          return (
            <Card key={plan.code} className="gap-0 py-0 shadow-xs">
              <CardContent className="flex h-full flex-col gap-4 p-4">
                <div>
                  <div className="flex items-center justify-between gap-3">
                    <h3 className="truncate text-base font-semibold text-foreground">{plan.name}</h3>
                    <Badge variant="outline">{plan.code}</Badge>
                  </div>
                  <p className="mt-2 line-clamp-2 min-h-10 text-xs leading-5 text-muted-foreground">{plan.description}</p>
                </div>
                <div className="mt-auto space-y-2">
                  <div className="text-sm font-semibold">
                    {price ? t("price", { price: amount, interval }) : t("empty")}
                  </div>
                  <div className="inline-flex items-center gap-2 text-xs text-muted-foreground">
                    <Check className="size-3.5 text-primary" strokeWidth={1.8} />
                    <span>{t("monthlyCredit", { credit: formatUSD(plan.periodCreditUSD, locale, { maximumFractionDigits: 4 }) })}</span>
                  </div>
                </div>
              </CardContent>
            </Card>
          );
        }) : (
          <div className="rounded-xl border border-border bg-card p-4 text-sm text-muted-foreground md:col-span-3">
            {t("empty")}
          </div>
        )}
      </div>
    </section>
  );
}

export function PublicLandingPage() {
  const t = useTranslations("landing");
  const [models, setModels] = React.useState<PublicModelDTO[]>([]);
  const [plans, setPlans] = React.useState<BillingPlanDTO[]>([]);
  const [loading, setLoading] = React.useState(true);

  React.useEffect(() => {
    let mounted = true;
    Promise.allSettled([listPublicModelCatalog(), listPublicBillingPlans()])
      .then(([modelResult, planResult]) => {
        if (!mounted) return;
        if (modelResult.status === "fulfilled") setModels(modelResult.value);
        if (planResult.status === "fulfilled") setPlans(planResult.value);
      })
      .finally(() => {
        if (mounted) setLoading(false);
      });
    return () => {
      mounted = false;
    };
  }, []);

  return (
    <main className="min-h-dvh overflow-y-auto bg-background text-foreground">
      <div className="mx-auto flex w-full max-w-6xl flex-col gap-12 px-4 py-5 md:gap-16 md:px-6 md:py-8">
        <header className="flex items-center justify-between gap-4">
          <Link href="/" className="flex min-w-0 items-center gap-3">
            <AppLogo width={96} height={40} priority className="h-7 w-auto object-contain" />
          </Link>
          <nav className="flex items-center gap-2">
            <Button variant="ghost" size="sm" className="relative after:absolute after:-inset-1 after:content-['']" asChild>
              <Link href="/status">{t("nav.status")}</Link>
            </Button>
            <Button variant="outline" size="sm" className="relative after:absolute after:-inset-1 after:content-['']" asChild>
              <Link href="/login">{t("nav.signIn")}</Link>
            </Button>
          </nav>
        </header>

        <section className="grid items-center gap-8 md:min-h-[min(680px,calc(100dvh-8rem))] lg:grid-cols-[minmax(0,1fr)_minmax(320px,0.78fr)]">
          <div className="space-y-6">
            <Badge variant="secondary" className="rounded-full">
              {t("hero.eyebrow")}
            </Badge>
            <div className="space-y-4">
              <h1 className="max-w-3xl text-3xl font-semibold tracking-tight md:text-6xl">
                {t("hero.title")}
              </h1>
              <p className="max-w-2xl text-sm leading-6 text-muted-foreground md:text-base md:leading-7">
                {t("hero.description")}
              </p>
            </div>
            <div className="flex flex-col gap-2 sm:flex-row">
              <Button size="lg" asChild>
                <Link href="/chat">
                  {t("hero.primary")}
                  <ArrowRight className="size-4" strokeWidth={1.7} />
                </Link>
              </Button>
              <Button size="lg" variant="outline" asChild>
                <Link href="/status">{t("hero.secondary")}</Link>
              </Button>
            </div>
            <div className="grid max-w-xl grid-cols-3 gap-2">
              <StatCard label={t("stats.models")} value={loading ? t("common.loading") : String(models.length)} />
              <StatCard label={t("stats.plans")} value={loading ? t("common.loading") : String(plans.length)} />
              <StatCard label={t("stats.mobile")} value={t("stats.mobileValue")} />
            </div>
          </div>

          <div className="rounded-2xl border border-border bg-card/80 p-3 shadow-sm">
            <div className="rounded-xl border border-border bg-background p-3">
              <div className="mb-3 flex items-center justify-between gap-3">
                <div className="flex items-center gap-2 text-sm font-medium">
                  <MessageSquareText className="size-4 text-primary" strokeWidth={1.7} />
                  <span>{t("preview.title")}</span>
                </div>
                <Badge variant="outline">{models.length || t("common.loading")}</Badge>
              </div>
              <div className="space-y-2">
                <div className="rounded-lg border border-border bg-muted/20 p-3">
                  <div className="text-xs font-medium text-foreground">{t("preview.question")}</div>
                  <div className="mt-1 text-xs leading-5 text-muted-foreground">{t("preview.answer")}</div>
                </div>
                {HERO_PREVIEW_KEYS.map((key) => (
                  <div key={key} className={cn("rounded-lg border border-border bg-muted/20 p-3", key === "projects" && "ml-5")}>
                    <div className="text-xs font-medium text-foreground">{t(`features.items.${key}.title`)}</div>
                    <div className="mt-1 text-xs leading-5 text-muted-foreground">{t(`features.items.${key}.description`)}</div>
                  </div>
                ))}
                <div className="rounded-lg border border-border bg-primary/10 p-3 text-xs leading-5 text-primary">
                  {t("preview.note")}
                </div>
              </div>
            </div>
          </div>
        </section>

        <section className="space-y-4">
          <div>
            <h2 className="text-xl font-semibold tracking-tight md:text-2xl">{t("features.title")}</h2>
            <p className="mt-1 max-w-2xl text-sm text-muted-foreground">{t("features.description")}</p>
          </div>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            {(["models", "projects", "sharing", "mobile"] as const).map((featureKey) => (
              <FeatureCard key={featureKey} featureKey={featureKey} />
            ))}
          </div>
        </section>

        <ModelCatalog models={models} loading={loading} />
        <PlansPreview plans={plans} loading={loading} />

        <section className="rounded-2xl border border-border bg-card px-4 py-6 text-card-foreground shadow-xs md:px-6">
          <div className="flex flex-col gap-4 md:flex-row md:items-center md:justify-between">
            <div className="space-y-1">
              <h2 className="text-xl font-semibold tracking-tight">{t("cta.title")}</h2>
              <p className="text-sm text-muted-foreground">{t("cta.description")}</p>
            </div>
            <Button asChild>
              <Link href="/chat">{t("cta.button")}</Link>
            </Button>
          </div>
        </section>
      </div>
    </main>
  );
}
