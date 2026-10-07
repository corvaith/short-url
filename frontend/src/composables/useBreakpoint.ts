import { computed, reactive } from 'vue'
import { BREAKPOINTS, BREAKPOINT_QUERIES, type BreakpointKey } from '../assets/breakpoints'

interface BreakpointFlags {
  base: boolean
  sm: boolean
  md: boolean
  lg: boolean
  xl: boolean
}

const flags = reactive<BreakpointFlags>({ base: true, sm: false, md: false, lg: false, xl: false })

let attached = false

function attach(): void {
  if (attached || typeof window === 'undefined') return
  attached = true
  const keys: Exclude<BreakpointKey, 'base'>[] = ['sm', 'md', 'lg', 'xl']
  for (const key of keys) {
    const mql = window.matchMedia(BREAKPOINT_QUERIES[key])
    const update = (): void => {
      flags[key] = mql.matches
    }
    update()
    mql.addEventListener('change', update)
  }
}

export function useBreakpoint() {
  attach()
  return {
    flags,
    isMobile: computed(() => !flags.md),
    isTablet: computed(() => flags.md && !flags.xl),
    isDesktop: computed(() => flags.xl),
    /** List switches from cards to a table at >=1280px. */
    isTableMode: computed(() => flags.xl),
  }
}

export { BREAKPOINTS }
