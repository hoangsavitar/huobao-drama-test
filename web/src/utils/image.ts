/**
 * Image URL helper functions.
 */

/**
 * Normalize image URL for relative/absolute inputs.
 */
export function fixImageUrl(url: string): string {
  if (!url) return "";
  if (url.startsWith("http") || url.startsWith("data:")) return url;
  return `${import.meta.env.VITE_API_BASE_URL || ""}${url}`;
}

/**
 * Get image URL, prefer local_path.
 * @param item object containing local_path or image_url
 * @returns normalized image URL
 */
export function getImageUrl(item: any): string {
  if (!item) return "";

  if (item.local_path) {
    return `/static/${item.local_path}`;
  }

  if (item.image_url) {
    return fixImageUrl(item.image_url);
  }

  return "";
}

/**
 * Check whether image exists.
 */
export function hasImage(item: any): boolean {
  return !!(item?.local_path || item?.image_url);
}

/**
 * Get video URL, prefer local_path.
 * @param item object containing local_path or video_url or url
 * @returns normalized video URL
 */
export function getVideoUrl(item: any): string {
  if (!item) return "";

  if (item.local_path) {
    if (item.local_path.startsWith("http")) {
      return item.local_path;
    }
    return `/static/${item.local_path}`;
  }

  if (item.video_url) {
    return fixImageUrl(item.video_url);
  }

  if (item.url) {
    return fixImageUrl(item.url);
  }

  return "";
}

/**
 * Check whether video exists.
 */
export function hasVideo(item: any): boolean {
  return !!(item?.local_path || item?.video_url || item?.url);
}
