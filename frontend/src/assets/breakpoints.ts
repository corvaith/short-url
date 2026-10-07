/** Named breakpoints, mobile-first (min-width), in px. */
export const BREAKPOINTS = {
  base: 0,
  sm: 480,
  md: 768,
  lg: 1024,
  xl: 1280,
} as const

export type BreakpointKey = keyof typeof BREAKPOINTS

/** MediaQueryList query strings derived from BREAKPOINTS. */
export const BREAKPOINT_QUERIES: Record<Exclude<BreakpointKey, 'base'>, string> = {
  sm: `(min-width: ${BREAKPOINTS.sm}px)`,
  md: `(min-width: ${BREAKPOINTS.md}px)`,
  lg: `(min-width: ${BREAKPOINTS.lg}px)`,
  xl: `(min-width: ${BREAKPOINTS.xl}px)`,
}
