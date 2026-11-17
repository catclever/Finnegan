/**
 * Finnegan Web Capture URL Detection Helper
 * Determines whether user input is a standalone single HTTP/HTTPS URL
 * without attached images or extraneous non-URL content.
 */

export const SINGLE_URL_REGEX = /^https?:\/\/[^\s]+$/i;

export function isSingleWebURL(text: string, attachedImagesCount: number = 0): boolean {
  if (attachedImagesCount > 0) return false;
  const trimmed = text.trim();
  return SINGLE_URL_REGEX.test(trimmed);
}

export function prependCapturedCardRef(
  text: string,
  capturedCard: { id: string; title: string; type: string } | null
): string {
  const trimmed = text.trim();
  if (!capturedCard || !capturedCard.title || !trimmed) return trimmed;
  const refToken = `@${capturedCard.title}`;
  if (trimmed.includes(refToken)) return trimmed;
  return `${refToken} ${trimmed}`;
}
