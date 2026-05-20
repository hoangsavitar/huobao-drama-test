import { FFmpeg } from '@ffmpeg/ffmpeg'
import { fetchFile, toBlobURL } from '@ffmpeg/util'

export interface VideoClip {
  url: string
  startTime: number
  endTime: number
  duration: number
  transition?: TransitionEffect
}

export type TransitionType = 'fade' | 'fadeblack' | 'fadewhite' | 'slideleft' | 'slideright' | 'slideup' | 'slidedown' | 'wipeleft' | 'wiperight' | 'circleopen' | 'circleclose' | 'none'

export interface TransitionEffect {
  type: TransitionType
  duration: number // Transition duration (seconds)
}

export interface MergeProgress {
  phase: 'loading' | 'processing' | 'encoding' | 'completed'
  progress: number
  message: string
}

class VideoMerger {
  private ffmpeg: FFmpeg
  private loaded: boolean = false
  private onProgress?: (progress: MergeProgress) => void

  constructor() {
    this.ffmpeg = new FFmpeg()
  }

  async initialize(onProgress?: (progress: MergeProgress) => void) {
    if (this.loaded) return

    this.onProgress = onProgress
    
    this.onProgress?.({
      phase: 'loading',
      progress: 0,
      message: 'Loading FFmpeg engine (first time requires ~30MB download)...'
    })

    // CDN list (ordered by preference)
    const cdnList = [
      'https://unpkg.zhimg.com/@ffmpeg/core@0.12.6/dist/esm',
      'https://npm.elemecdn.com/@ffmpeg/core@0.12.6/dist/esm',
      'https://cdn.jsdelivr.net/npm/@ffmpeg/core@0.12.6/dist/esm',
      'https://unpkg.com/@ffmpeg/core@0.12.6/dist/esm',
    ]
    
    this.ffmpeg.on('log', ({ message }) => {
      console.log('[FFmpeg]', message)
    })

    this.ffmpeg.on('progress', ({ progress, time }) => {
      this.onProgress?.({
        phase: 'encoding',
        progress: Math.round(progress * 100),
        message: `Merging video... ${Math.round(progress * 100)}%`
      })
    })

    // Try multiple CDN sources
    let lastError: Error | null = null
    for (let i = 0; i < cdnList.length; i++) {
      const baseURL = cdnList[i]
      
      try {
        this.onProgress?.({
          phase: 'loading',
          progress: (i / cdnList.length) * 50,
          message: `Loading FFmpeg from CDN ${i + 1}/${cdnList.length}...`
        })

        // Add timeout control
        const loadPromise = this.ffmpeg.load({
          coreURL: await toBlobURL(`${baseURL}/ffmpeg-core.js`, 'text/javascript'),
          wasmURL: await toBlobURL(`${baseURL}/ffmpeg-core.wasm`, 'application/wasm'),
        })

        const timeoutPromise = new Promise((_, reject) => {
          setTimeout(() => reject(new Error('Load timeout')), 60000) // 60 second timeout
        })

        await Promise.race([loadPromise, timeoutPromise])
        
        this.loaded = true
        
        this.onProgress?.({
          phase: 'loading',
          progress: 100,
          message: 'FFmpeg loaded successfully'
        })
        
        return
      } catch (error) {
        console.error(`CDN ${i + 1} failed:`, error)
        lastError = error as Error
        
        if (i < cdnList.length - 1) {
          this.onProgress?.({
            phase: 'loading',
            progress: ((i + 1) / cdnList.length) * 50,
            message: `CDN ${i + 1} failed, trying fallback...`
          })
        }
      }
    }

    // All CDN sources failed
    throw new Error(`FFmpeg load failed: ${lastError?.message || 'Unknown error'}. Please check your network connection and try again.`)
  }

  async mergeVideos(clips: VideoClip[]): Promise<Blob> {
    if (!this.loaded) {
      await this.initialize(this.onProgress)
    }

    if (clips.length === 0) {
      throw new Error('No video clips provided')
    }

    this.onProgress?.({
      phase: 'processing',
      progress: 0,
      message: 'Downloading video clips...'
    })

    this.onProgress?.({
      phase: 'processing',
      progress: 0,
      message: `Downloading ${clips.length} video clips...`
    })

    const downloadPromises = clips.map((clip, i) => 
      fetchFile(clip.url).then(data => ({ index: i, data }))
    )
    
    const downloads = await Promise.all(downloadPromises)
    
    this.onProgress?.({
      phase: 'processing',
      progress: 30,
      message: 'Download complete, processing video...'
    })

    const inputFiles: string[] = []
    for (let i = 0; i < clips.length; i++) {
      const clip = clips[i]
      const download = downloads.find(d => d.index === i)!
      const inputFileName = `input${i}.mp4`
      const outputFileName = `clip${i}.mp4`
      
      await this.ffmpeg.writeFile(inputFileName, download.data)

      if (clip.startTime > 0 || clip.endTime < clip.duration) {
        this.onProgress?.({
          phase: 'processing',
          progress: Math.round(30 + (i / clips.length) * 20),
          message: `Trimming clip ${i + 1}/${clips.length}...`
        })

        await this.ffmpeg.exec([
          '-i', inputFileName,
          '-ss', clip.startTime.toString(),
          '-t', (clip.endTime - clip.startTime).toString(),
          '-c', 'copy',
          outputFileName
        ])
        
        inputFiles.push(outputFileName)
        await this.ffmpeg.deleteFile(inputFileName)
      } else {
        inputFiles.push(inputFileName)
      }
    }

    this.onProgress?.({
      phase: 'processing',
      progress: 50,
      message: 'Preparing to merge...'
    })

    const hasTransitions = clips.some(clip => clip.transition && clip.transition.type !== 'none')

    if (!hasTransitions || clips.length === 1) {
      const concatContent = inputFiles.map(f => `file '${f}'`).join('\n')
      await this.ffmpeg.writeFile('concat.txt', concatContent)

      this.onProgress?.({
        phase: 'encoding',
        progress: 0,
        message: 'Merging video...'
      })

      await this.ffmpeg.exec([
        '-f', 'concat',
        '-safe', '0',
        '-i', 'concat.txt',
        '-c', 'copy',
        '-movflags', '+faststart',
        'output.mp4'
      ])
    } else {
      this.onProgress?.({
        phase: 'encoding',
        progress: 0,
        message: 'Applying transitions and merging video (this may take a while)...'
      })

      await this.mergeWithTransitions(inputFiles, clips)
    }

    this.onProgress?.({
      phase: 'completed',
      progress: 90,
      message: 'Generating final file...'
    })

    const data = await this.ffmpeg.readFile('output.mp4')
    const blob = new Blob([data], { type: 'video/mp4' })

    for (const file of inputFiles) {
      await this.ffmpeg.deleteFile(file)
    }
    await this.ffmpeg.deleteFile('concat.txt')
    await this.ffmpeg.deleteFile('output.mp4')

    this.onProgress?.({
      phase: 'completed',
      progress: 100,
      message: 'Merge complete!'
    })

    return blob
  }

  private async mergeWithTransitions(inputFiles: string[], clips: VideoClip[]) {
    const filterParts: string[] = []
    const inputs: string[] = []
    
    for (let i = 0; i < inputFiles.length; i++) {
      inputs.push('-i', inputFiles[i])
      filterParts.push(`[${i}:v]setpts=PTS-STARTPTS[v${i}]`)
      filterParts.push(`[${i}:a]asetpts=PTS-STARTPTS[a${i}]`)
    }
    
    let videoChain = 'v0'
    let audioChain = 'a0'
    
    for (let i = 1; i < clips.length; i++) {
      const transition = clips[i].transition
      const transType = transition?.type || 'fade'
      const transDuration = transition?.duration || 1.0
      
      const offset = clips.slice(0, i).reduce((sum, c) => sum + c.duration, 0) - transDuration
      
      const xfadeFilter = this.getXfadeFilter(transType, transDuration, offset)
      filterParts.push(`[${videoChain}][v${i}]${xfadeFilter}[v${i}out]`)
      videoChain = `v${i}out`
      
      filterParts.push(`[${audioChain}][a${i}]acrossfade=d=${transDuration}:c1=tri:c2=tri[a${i}out]`)
      audioChain = `a${i}out`
    }
    
    const filterComplex = filterParts.join(';')
    
    await this.ffmpeg.exec([
      ...inputs,
      '-filter_complex', filterComplex,
      '-map', `[${videoChain}]`,
      '-map', `[${audioChain}]`,
      '-c:v', 'libx264',
      '-preset', 'ultrafast',
      '-crf', '23',
      '-c:a', 'aac',
      '-b:a', '128k',
      '-movflags', '+faststart',
      'output.mp4'
    ])
  }
  
  private getXfadeFilter(type: TransitionType, duration: number, offset: number): string {
    const xfadeTypes: Record<string, string> = {
      'fade': 'fade',
      'fadeblack': 'fadeblack',
      'fadewhite': 'fadewhite',
      'slideleft': 'slideleft',
      'slideright': 'slideright',
      'slideup': 'slideup',
      'slidedown': 'slidedown',
      'wipeleft': 'wipeleft',
      'wiperight': 'wiperight',
      'circleopen': 'circleopen',
      'circleclose': 'circleclose'
    }
    
    const xfadeType = xfadeTypes[type] || 'fade'
    return `xfade=transition=${xfadeType}:duration=${duration}:offset=${offset}`
  }

  async terminate() {
    if (this.loaded) {
      this.ffmpeg.terminate()
      this.loaded = false
    }
  }
}

export const videoMerger = new VideoMerger()
