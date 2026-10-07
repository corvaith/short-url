import { onMounted } from 'vue'
import { APP_NAME } from '../constants'

interface PageMetaOptions {
  /** Public pages are indexable; everything else gets robots noindex. */
  public?: boolean
}

export function usePageMeta(title: string, opts: PageMetaOptions = {}): void {
  onMounted(() => {
    document.title = `${title} — ${APP_NAME}`
    const existing = document.querySelector('meta[name="robots"]')
    if (!opts.public) {
      let robots = existing
      if (!robots) {
        robots = document.createElement('meta')
        robots.setAttribute('name', 'robots')
        document.head.appendChild(robots)
      }
      robots.setAttribute('content', 'noindex')
    } else if (existing) {
      existing.remove()
    }
  })
}
