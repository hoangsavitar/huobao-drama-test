import JSZip from 'jszip'
import type { ImageGeneration } from '@/types/image'
import { getImageUrl } from '@/utils/image'

function sanitizeSegment(name: string, fallback: string): string {
  const t = (name || '')
    .replace(/[/\\:*?"<>|]/g, '_')
    .replace(/\s+/g, ' ')
    .trim()
    .slice(0, 100)
  return t || fallback
}

function extensionFromUrl(url: string): string {
  const u = url.split('?')[0].toLowerCase()
  const m = u.match(/\.(jpe?g|png|webp|gif)$/)
  if (!m) return '.jpg'
  const e = m[1]
  return `.${e === 'jpeg' ? 'jpg' : e}`
}

function toFetchableUrl(pathOrUrl: string): string {
  if (!pathOrUrl) return ''
  if (pathOrUrl.startsWith('data:') || pathOrUrl.startsWith('http')) return pathOrUrl
  const path = pathOrUrl.startsWith('/') ? pathOrUrl : `/${pathOrUrl}`
  if (typeof window === 'undefined') return path
  return `${window.location.origin}${path}`
}

async function fetchImageBytes(url: string): Promise<Uint8Array | null> {
  try {
    if (url.startsWith('data:')) {
      const comma = url.indexOf(',')
      const b64 = url.slice(comma + 1)
      const bin = atob(b64)
      const out = new Uint8Array(bin.length)
      for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i)
      return out
    }
    const res = await fetch(url, { credentials: 'same-origin' })
    if (!res.ok) return null
    return new Uint8Array(await res.arrayBuffer())
  } catch {
    return null
  }
}

function normalizeUrlKey(u: string): string {
  return u.split('?')[0].trim()
}

export interface ShotRow {
  id: number | string
  storyboard_number?: number
  title?: string
  composed_image?: string
}

export interface BuildShotImagesZipOptions {
  dramaTitle: string
  episodeNumber: number
  episodeTitle: string
  shots: ShotRow[]
  /** Completed storyboard image generations for this drama */
  imageGens: ImageGeneration[]
}

/**
 * Zip layout:
 *   Ep{n}_{episodeTitle}/{dramaTitle}/shot_{n}_{title}/img_{id}_{frame}.ext
 *   + composed.ext when storyboard has composed_image and it is not duplicate of a gen file
 */
export async function buildShotImagesZip(
  opts: BuildShotImagesZipOptions
): Promise<Blob> {
  const zip = new JSZip()
  const epSeg = sanitizeSegment(
    `Ep${opts.episodeNumber}_${opts.episodeTitle || 'episode'}`,
    `Ep${opts.episodeNumber}`
  )
  const dramaSeg = sanitizeSegment(opts.dramaTitle || 'drama', 'drama')
  const base = `${epSeg}/${dramaSeg}`

  const shotIdSet = new Set(
    opts.shots.map((s) => Number(s.id)).filter((id) => !Number.isNaN(id))
  )

  const gensByStoryboard = new Map<number, ImageGeneration[]>()
  for (const g of opts.imageGens) {
    if (g.image_type && g.image_type !== "storyboard") continue
    const sid = g.storyboard_id
    if (sid == null || !shotIdSet.has(sid)) continue
    const list = gensByStoryboard.get(sid) || []
    list.push(g)
    gensByStoryboard.set(sid, list)
  }

  for (const [, list] of gensByStoryboard) {
    list.sort((a, b) => a.id - b.id)
  }

  let addedAny = false

  for (const shot of opts.shots) {
    const sid = Number(shot.id)
    if (Number.isNaN(sid)) continue

    const num = shot.storyboard_number ?? sid
    const titlePart = shot.title ? `_${sanitizeSegment(shot.title, '')}` : ''
    const shotFolder = sanitizeSegment(`shot_${num}${titlePart}`, `shot_${num}`)
    const prefix = `${base}/${shotFolder}/`

    const usedUrlKeys = new Set<string>()
    const gens = gensByStoryboard.get(sid) || []

    for (const g of gens) {
      const rel = getImageUrl({
        local_path: g.local_path,
        image_url: g.image_url,
      })
      if (!rel) continue
      const fetchUrl = toFetchableUrl(rel)
      const bytes = await fetchImageBytes(fetchUrl)
      if (!bytes) continue
      const ext = extensionFromUrl((g.image_url || rel) as string)
      const ft = (g.frame_type || 'frame').replace(/[/\\:*?"<>|]/g, '_')
      const name = `img_${g.id}_${ft}${ext}`
      zip.file(`${prefix}${name}`, bytes)
      usedUrlKeys.add(normalizeUrlKey(fetchUrl))
      if (g.image_url) usedUrlKeys.add(normalizeUrlKey(g.image_url))
      addedAny = true
    }

    const comp = shot.composed_image?.trim()
    if (comp) {
      const compFetch = comp.startsWith('http') || comp.startsWith('data:')
        ? comp
        : toFetchableUrl(comp.startsWith('/') ? comp : `/${comp}`)
      const key = normalizeUrlKey(compFetch)
      if (!usedUrlKeys.has(key)) {
        const bytes = await fetchImageBytes(compFetch)
        if (bytes) {
          zip.file(
            `${prefix}composed${extensionFromUrl(comp)}`,
            bytes
          )
          addedAny = true
        }
      }
    }
  }

  if (!addedAny) {
    zip.file(
      `${base}/README.txt`,
      'No image files could be downloaded for this episode. Check that images are completed and URLs are reachable from the browser.'
    )
  }

  return zip.generateAsync({ type: 'blob', compression: 'DEFLATE' })
}
