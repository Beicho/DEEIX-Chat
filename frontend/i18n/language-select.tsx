"use client";

import * as React from "react";
import { useTranslations } from "next-intl";

import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { APP_LOCALE_LABELS, APP_LOCALES, type AppLocale } from "@/i18n/config";
import { useAppLocale } from "@/i18n/app-i18n-provider";
import { cn } from "@/lib/utils";

export function LanguageSelect({
  className,
  disabled,
  onValueChange,
  triggerClassName,
  value,
}: {
  className?: string;
  disabled?: boolean;
  onValueChange?: (locale: AppLocale) => void;
  triggerClassName?: string;
  value?: AppLocale;
}) {
  const t = useTranslations("common.locale");
  const { locale, setLocale } = useAppLocale();
  const selectedLocale = value ?? locale;

  return (
    <div className={cn("min-w-0", className)}>
      <Select
        value={selectedLocale}
        onValueChange={(value) => {
          const nextLocale = value as AppLocale;
          onValueChange?.(nextLocale);
          void setLocale(nextLocale);
        }}
        disabled={disabled}
      >
        <SelectTrigger aria-label={t("label")} className={cn("h-8 w-[8.25rem]", triggerClassName)}>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {APP_LOCALES.map((item) => (
            <SelectItem key={item} value={item}>
              {APP_LOCALE_LABELS[item]}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
