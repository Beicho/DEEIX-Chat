"use client"

import { CircleAlert } from "lucide-react"
import { useTranslations } from "next-intl"
import { useEffect } from "react"

import { Button } from "@/components/ui/button"

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}) {
  const t = useTranslations("common.error")

  useEffect(() => {
    console.error(error)
  }, [error])

  return (
    <div className="flex min-h-dvh w-full flex-col items-center justify-center gap-6 bg-background px-4 py-12 text-center">
      <div className="flex flex-col items-center gap-4">
        <CircleAlert className="size-12 text-destructive" aria-hidden="true" />
        <div className="flex flex-col gap-2">
          <h1 className="text-xl font-semibold text-foreground">
            {t("errorTitle")}
          </h1>
          <p className="max-w-md text-sm text-muted-foreground">
            {t("errorDescription")}
          </p>
        </div>
      </div>
      <Button
        variant="default"
        size="lg"
        onClick={reset}
        className="min-h-11 px-6"
      >
        {t("retry")}
      </Button>
    </div>
  )
}
