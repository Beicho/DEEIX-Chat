"use client";

import * as React from "react";
import { createPortal } from "react-dom";
import Image from "next/image";
import { Maximize2, Minus, Plus, X } from "lucide-react";
import { useTranslations } from "next-intl";

import { Button } from "@/components/ui/button";
import {
  IMAGE_PREVIEW_DEFAULT_ZOOM,
  IMAGE_PREVIEW_MAX_ZOOM,
  IMAGE_PREVIEW_MIN_ZOOM,
  clampImagePreviewZoom,
  getPointerDistance,
  resolveDoubleTapImageZoom,
  resolvePinchImageZoom,
  type Point,
} from "@/features/files/model/image-preview-zoom";

type ImagePreviewOverlayProps = {
  open: boolean;
  source: string;
  alt?: string;
  contentType?: string;
  onOpenChange: (open: boolean) => void;
};

const IMAGE_ZOOM_STEP = 0.1;
const IMAGE_FALLBACK_SIZE = {
  width: 1200,
  height: 900,
};

function resolveImageSize(width: number, height: number) {
  return {
    width: width || IMAGE_FALLBACK_SIZE.width,
    height: height || IMAGE_FALLBACK_SIZE.height,
  };
}

export function ImagePreviewOverlay({ open, source, alt, contentType, onOpenChange }: ImagePreviewOverlayProps) {
  const t = useTranslations("files.previewDialog");
  const scrollRegionRef = React.useRef<HTMLDivElement | null>(null);
  const pointersRef = React.useRef<Map<number, Point>>(new Map());
  const pinchRef = React.useRef<{ startDistance: number; startZoom: number } | null>(null);
  const [mounted, setMounted] = React.useState(false);
  const [zoom, setZoom] = React.useState(IMAGE_PREVIEW_DEFAULT_ZOOM);
  const [imageSize, setImageSize] = React.useState(IMAGE_FALLBACK_SIZE);
  const isSVG = contentType?.split(";")[0]?.trim().toLowerCase() === "image/svg+xml";

  React.useEffect(() => {
    setMounted(true);
  }, []);

  React.useEffect(() => {
    if (!open) {
      return;
    }
    setZoom(IMAGE_PREVIEW_DEFAULT_ZOOM);
    pointersRef.current.clear();
    pinchRef.current = null;
  }, [open, source]);

  React.useEffect(() => {
    if (!open) {
      return undefined;
    }

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onOpenChange(false);
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [onOpenChange, open]);

  if (!mounted || !open || !source) {
    return null;
  }

  const changeZoom = (delta: number) => {
    setZoom((value) => clampImagePreviewZoom(value + delta));
  };

  const handlePointerDown = (event: React.PointerEvent<HTMLDivElement>) => {
    event.currentTarget.setPointerCapture(event.pointerId);
    pointersRef.current.set(event.pointerId, { x: event.clientX, y: event.clientY });
    const pointers = Array.from(pointersRef.current.values());
    if (pointers.length === 2) {
      pinchRef.current = {
        startDistance: getPointerDistance(pointers[0], pointers[1]),
        startZoom: zoom,
      };
    }
  };

  const handlePointerMove = (event: React.PointerEvent<HTMLDivElement>) => {
    if (!pointersRef.current.has(event.pointerId)) {
      return;
    }
    pointersRef.current.set(event.pointerId, { x: event.clientX, y: event.clientY });
    const pointers = Array.from(pointersRef.current.values());
    if (pointers.length !== 2 || !pinchRef.current) {
      return;
    }
    event.preventDefault();
    setZoom(resolvePinchImageZoom({
      ...pinchRef.current,
      currentDistance: getPointerDistance(pointers[0], pointers[1]),
    }));
  };

  const handlePointerEnd = (event: React.PointerEvent<HTMLDivElement>) => {
    pointersRef.current.delete(event.pointerId);
    if (pointersRef.current.size < 2) {
      pinchRef.current = null;
    }
  };

  const overlay = (
    <div
      className="fixed inset-0 z-[120] flex flex-col bg-background/96 text-foreground backdrop-blur"
      role="dialog"
      aria-modal="true"
      aria-label={alt?.trim() || t("imagePreview")}
    >
      <div className="flex min-h-14 shrink-0 items-center justify-between gap-3 border-b border-border/60 px-3 py-2">
        <p className="min-w-0 truncate text-sm font-medium">{alt?.trim() || t("imagePreview")}</p>
        <div className="flex shrink-0 items-center gap-1">
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-10 rounded-full md:size-8"
            aria-label={t("zoomOut")}
            onClick={() => changeZoom(-IMAGE_ZOOM_STEP)}
            disabled={zoom <= IMAGE_PREVIEW_MIN_ZOOM}
          >
            <Minus className="size-4" />
          </Button>
          <span className="min-w-12 text-center text-[11px] text-muted-foreground">{Math.round(zoom * 100)}%</span>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-10 rounded-full md:size-8"
            aria-label={t("zoomIn")}
            onClick={() => changeZoom(IMAGE_ZOOM_STEP)}
            disabled={zoom >= IMAGE_PREVIEW_MAX_ZOOM}
          >
            <Plus className="size-4" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-10 rounded-full md:size-8"
            aria-label={t("fitImage")}
            onClick={() => setZoom(IMAGE_PREVIEW_DEFAULT_ZOOM)}
          >
            <Maximize2 className="size-4" />
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-10 rounded-full md:size-8"
            aria-label={t("closePreview")}
            onClick={() => onOpenChange(false)}
          >
            <X className="size-4" />
          </Button>
        </div>
      </div>
      <div
        ref={scrollRegionRef}
        className="min-h-0 flex-1 touch-none overflow-auto overscroll-contain"
        onDoubleClick={() => setZoom((value) => resolveDoubleTapImageZoom(value))}
        onPointerCancel={handlePointerEnd}
        onPointerDown={handlePointerDown}
        onPointerMove={handlePointerMove}
        onPointerUp={handlePointerEnd}
        onWheel={(event) => {
          if (!event.ctrlKey && !event.metaKey) {
            return;
          }
          event.preventDefault();
          changeZoom(event.deltaY > 0 ? -IMAGE_ZOOM_STEP : IMAGE_ZOOM_STEP);
        }}
      >
        <div className="flex min-h-full min-w-full items-center justify-center p-4">
          <div
            className="relative shrink-0"
            style={{
              width: `${imageSize.width * zoom}px`,
              height: `${imageSize.height * zoom}px`,
            }}
          >
            <div
              style={{
                width: `${imageSize.width}px`,
                height: `${imageSize.height}px`,
                transform: `scale(${zoom})`,
                transformOrigin: "top left",
              }}
            >
              {isSVG ? (
                <object
                  data={source}
                  type="image/svg+xml"
                  aria-label={alt || t("imagePreview")}
                  className="block rounded-lg"
                  style={{ width: `${imageSize.width}px`, height: `${imageSize.height}px` }}
                >
                  <Image
                    src={source}
                    alt={alt || t("imagePreview")}
                    className="block rounded-lg object-contain"
                    width={imageSize.width}
                    height={imageSize.height}
                    sizes="100vw"
                    unoptimized
                  />
                </object>
              ) : (
                <Image
                  src={source}
                  alt={alt || t("imagePreview")}
                  className="block rounded-lg object-contain"
                  width={imageSize.width}
                  height={imageSize.height}
                  sizes="100vw"
                  unoptimized
                  onLoad={(event) => {
                    setImageSize(resolveImageSize(event.currentTarget.naturalWidth, event.currentTarget.naturalHeight));
                  }}
                />
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );

  return createPortal(overlay, document.body);
}
