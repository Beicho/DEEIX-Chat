export const LONG_PASTE_TEXT_THRESHOLD = 4000;

function padDatePart(value: number, size = 2): string {
  return String(value).padStart(size, "0");
}

export function shouldConvertPasteToTextFile(text: string, threshold = LONG_PASTE_TEXT_THRESHOLD): boolean {
  return text.trim().length >= threshold;
}

export function buildPastedTextFileName(date = new Date()): string {
  const year = date.getUTCFullYear();
  const month = padDatePart(date.getUTCMonth() + 1);
  const day = padDatePart(date.getUTCDate());
  const hour = padDatePart(date.getUTCHours());
  const minute = padDatePart(date.getUTCMinutes());
  const second = padDatePart(date.getUTCSeconds());
  return `pasted-text-${year}${month}${day}-${hour}${minute}${second}.txt`;
}

export function createPastedTextFile(text: string, date = new Date()): File {
  return new File([text], buildPastedTextFileName(date), {
    type: "text/plain;charset=utf-8",
    lastModified: date.getTime(),
  });
}
