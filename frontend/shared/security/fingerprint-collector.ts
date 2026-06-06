import { sha256Hex } from "@/shared/security/encoding";

export type DeviceFingerprint = {
  fingerprintId?: string;
  screenResolution: string;
  colorDepth: number;
  pixelRatio: number;
  hardwareConcurrency: number;
  deviceMemory: number;
  maxTouchPoints: number;
  userAgent: string;
  language: string;
  timezone: string;
  platform: string;
  canvasHash: string;
  webglVendor: string;
  webglRenderer: string;
  fontsHash: string;
  audioHash: string;
};

let cachedFingerprint: Promise<DeviceFingerprint> | null = null;

export async function collectDeviceFingerprint(): Promise<DeviceFingerprint> {
  if (cachedFingerprint) {
    return cachedFingerprint;
  }
  cachedFingerprint = collectDeviceFingerprintFresh().catch((error) => {
    cachedFingerprint = null;
    throw error;
  });
  return cachedFingerprint;
}

async function collectDeviceFingerprintFresh(): Promise<DeviceFingerprint> {
  const fp: DeviceFingerprint = {
    screenResolution: safeScreenResolution(),
    colorDepth: safeNumber(() => window.screen.colorDepth),
    pixelRatio: safeNumber(() => window.devicePixelRatio),
    hardwareConcurrency: safeNumber(() => navigator.hardwareConcurrency),
    deviceMemory: safeNumber(() => (navigator as Navigator & { deviceMemory?: number }).deviceMemory),
    maxTouchPoints: safeNumber(() => navigator.maxTouchPoints),
    userAgent: navigator.userAgent || "",
    language: navigator.language || "",
    timezone: safeTimezone(),
    platform: navigator.platform || "",
    canvasHash: await canvasFingerprint(),
    webglVendor: "",
    webglRenderer: "",
    fontsHash: await fontsFingerprint(),
    audioHash: await audioFingerprint(),
  };
  const webgl = webglFingerprint();
  fp.webglVendor = webgl.vendor;
  fp.webglRenderer = webgl.renderer;
  fp.fingerprintId = await calculateFingerprintID(fp);
  return fp;
}

export async function calculateFingerprintID(fp: DeviceFingerprint): Promise<string> {
  return sha256Hex(stableFingerprintParts(fp).join("|"));
}

function stableFingerprintParts(fp: DeviceFingerprint): string[] {
  return [
    fp.screenResolution || "",
    intString(fp.colorDepth),
    floatString(fp.pixelRatio),
    intString(fp.hardwareConcurrency),
    intString(fp.deviceMemory),
    intString(fp.maxTouchPoints),
    fp.language || "",
    fp.timezone || "",
    fp.platform || "",
    fp.canvasHash || "",
    fp.webglVendor || "",
    fp.webglRenderer || "",
    fp.fontsHash || "",
    fp.audioHash || "",
  ];
}

function safeScreenResolution(): string {
  try {
    return `${window.screen.width}x${window.screen.height}`;
  } catch {
    return "";
  }
}

function safeNumber(read: () => number | undefined): number {
  try {
    const value = read();
    return typeof value === "number" && Number.isFinite(value) ? value : 0;
  } catch {
    return 0;
  }
}

function safeTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "";
  } catch {
    return "";
  }
}

async function canvasFingerprint(): Promise<string> {
  try {
    const canvas = document.createElement("canvas");
    canvas.width = 240;
    canvas.height = 60;
    const ctx = canvas.getContext("2d");
    if (!ctx) {
      return "";
    }
    ctx.textBaseline = "top";
    ctx.font = "14px Arial";
    ctx.fillStyle = "#f60";
    ctx.fillRect(125, 1, 62, 20);
    ctx.fillStyle = "#069";
    ctx.fillText("DEEIX", 2, 15);
    ctx.fillStyle = "rgba(102, 204, 0, 0.7)";
    ctx.fillText("browser proof", 4, 34);
    return sha256Hex(canvas.toDataURL());
  } catch {
    return "";
  }
}

function webglFingerprint(): { vendor: string; renderer: string } {
  try {
    const canvas = document.createElement("canvas");
    const gl = (canvas.getContext("webgl") || canvas.getContext("experimental-webgl")) as WebGLRenderingContext | null;
    if (!gl) {
      return { vendor: "", renderer: "" };
    }
    const debugInfo = gl.getExtension("WEBGL_debug_renderer_info");
    if (!debugInfo) {
      return { vendor: "", renderer: "" };
    }
    return {
      vendor: String(gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL) || ""),
      renderer: String(gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL) || ""),
    };
  } catch {
    return { vendor: "", renderer: "" };
  }
}

async function fontsFingerprint(): Promise<string> {
  try {
    const fonts = ["Arial", "Courier New", "Georgia", "Helvetica", "Times New Roman", "Verdana", "PingFang SC", "Microsoft YaHei"];
    if (!document.fonts?.check) {
      return "";
    }
    const available = fonts.filter((font) => document.fonts.check(`12px "${font}"`));
    return sha256Hex(available.join("|"));
  } catch {
    return "";
  }
}

async function audioFingerprint(): Promise<string> {
  try {
    const AudioContextClass = window.OfflineAudioContext || (window as Window & { webkitOfflineAudioContext?: typeof OfflineAudioContext }).webkitOfflineAudioContext;
    if (!AudioContextClass) {
      return "";
    }
    const context = new AudioContextClass(1, 256, 44100);
    const oscillator = context.createOscillator();
    const compressor = context.createDynamicsCompressor();
    oscillator.type = "triangle";
    oscillator.frequency.value = 10000;
    compressor.threshold.value = -50;
    compressor.knee.value = 40;
    compressor.ratio.value = 12;
    compressor.attack.value = 0;
    compressor.release.value = 0.25;
    oscillator.connect(compressor);
    compressor.connect(context.destination);
    oscillator.start(0);
    const buffer = await context.startRendering();
    const samples = Array.from(buffer.getChannelData(0).slice(0, 64), (value) => value.toFixed(6)).join(",");
    return sha256Hex(samples);
  } catch {
    return "";
  }
}

function intString(value: number): string {
  return Number.isFinite(value) ? String(Math.trunc(value)) : "0";
}

function floatString(value: number): string {
  if (!Number.isFinite(value)) {
    return "0";
  }
  return value.toFixed(2).replace(/0+$/g, "").replace(/\.$/, "") || "0";
}

