import { useState, useEffect, useCallback, useRef } from 'react'

// Event types matching server-side events/types.go
export type EventType =
  // Session events
  | 'session:start'
  | 'session:end'
  | 'session:update'
  // Attention events
  | 'attention:warning'
  | 'attention:critical'
  | 'compact:triggered'
  | 'checkpoint:created'
  // Port events
  | 'port:start'
  | 'port:end'
  | 'port:blocked'
  // Checklist events
  | 'checklist:failed'
  | 'checklist:passed'
  // Escalation events
  | 'escalation:created'
  | 'escalation:resolved'
  // Message events
  | 'message:received'
  // Build events
  | 'build:failed'
  | 'test:failed'
  // Connection events
  | 'connection:established'

export interface SSEEvent<T = unknown> {
  type: EventType
  timestamp: string
  session_id?: string
  port_id?: string
  data?: T
}

export interface SessionStartData {
  id: string
  title?: string
  type?: string
  project_root?: string
}

export interface SessionEndData {
  id: string
  reason?: string
  status?: string
}

export interface AttentionWarningData {
  usage_percent: number
  tokens_used: number
  token_budget: number
  warning: string
}

export interface CompactTriggeredData {
  trigger: string
  checkpoint_id?: string
  recovery_hint?: string
}

export interface PortStartData {
  id: string
  title?: string
  checklist?: string[]
}

export interface PortEndData {
  id: string
  status: string
  duration_secs?: number
}

export interface ChecklistResultData {
  passed: boolean
  passed_count: number
  failed_count: number
  items: Array<{
    name: string
    passed: boolean
    message?: string
  }>
  blocked_by?: string[]
}

export interface BuildFailedData {
  fail_type: 'build' | 'test'
  exit_code: number
  error?: string
}

export interface UseSSEOptions {
  filters?: EventType[]
  sessionId?: string
  maxEvents?: number
  autoReconnect?: boolean
  reconnectDelay?: number
  baseUrl?: string
}

export interface UseSSEResult {
  events: SSEEvent[]
  latestEvent: SSEEvent | null
  connected: boolean
  error: Error | null
  clearEvents: () => void
  reconnect: () => void
}

const DEFAULT_BASE_URL = typeof window !== 'undefined' && window.location ? 
  `${window.location.protocol}//${window.location.host}` : 'http://localhost:9000'
const DEFAULT_MAX_EVENTS = 100
const DEFAULT_RECONNECT_DELAY = 3000

// Get server port from Electron if available
function getSSEBaseUrl(baseUrl: string): string {
  // In Electron, use the server port from main process
  if (typeof window !== 'undefined' && (window as any).app?.getServerPort) {
    // This is async but we need sync URL, so we'll handle it differently
    return baseUrl
  }
  // In web mode during dev, use Vite proxy (relative URL)
  if (import.meta.env.DEV) {
    return '' // Use relative URL, Vite will proxy
  }
  return baseUrl
}

export function useSSE(options: UseSSEOptions = {}): UseSSEResult {
  const {
    filters = [],
    sessionId,
    maxEvents = DEFAULT_MAX_EVENTS,
    autoReconnect = true,
    reconnectDelay = DEFAULT_RECONNECT_DELAY,
    baseUrl,
  } = options

  const [events, setEvents] = useState<SSEEvent[]>([])
  const [latestEvent, setLatestEvent] = useState<SSEEvent | null>(null)
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<Error | null>(null)
  // null = not resolved yet, string = resolved (empty string is valid for Vite proxy)
  const [resolvedBaseUrl, setResolvedBaseUrl] = useState<string | null>(baseUrl ?? null)

  const eventSourceRef = useRef<EventSource | null>(null)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const isResolvingRef = useRef(false)
  const isConnectedRef = useRef(false)

  // Resolve base URL (async for Electron)
  useEffect(() => {
    // Skip if already resolved or resolving
    if (resolvedBaseUrl !== null || isResolvingRef.current) {
      return
    }
    
    isResolvingRef.current = true
    
    async function resolveUrl() {
      try {
        // DEV mode: ALWAYS use Vite proxy (handles CORS)
        if (import.meta.env.DEV) {
          console.log('[SSE] Dev mode - using Vite proxy')
          setResolvedBaseUrl('')
          return
        }
        
        // Production Electron: use the Electron server port
        if (typeof window !== 'undefined' && (window as any).app?.isServerRunning) {
          const isRunning = await (window as any).app.isServerRunning()
          
          if (isRunning) {
            const port = await (window as any).app.getServerPort()
            if (port && port > 0) {
              console.log('[SSE] Using Electron server port:', port)
              setResolvedBaseUrl(`http://localhost:${port}`)
              return
            }
          }
        }
        
        // Production web: use current location (skip file:// protocol in Electron)
        if (DEFAULT_BASE_URL.startsWith('file://')) {
          console.log('[SSE] Electron file:// protocol detected, using localhost:9000 fallback')
          setResolvedBaseUrl('http://localhost:9000')
        } else {
          console.log('[SSE] Using fallback URL:', DEFAULT_BASE_URL)
          setResolvedBaseUrl(DEFAULT_BASE_URL)
        }
      } catch (e) {
        console.warn('[SSE] Failed to resolve URL:', e)
        setResolvedBaseUrl(import.meta.env.DEV ? '' : DEFAULT_BASE_URL)
      } finally {
        isResolvingRef.current = false
      }
    }
    
    resolveUrl()
  }, [baseUrl, resolvedBaseUrl])

  const clearEvents = useCallback(() => {
    setEvents([])
    setLatestEvent(null)
  }, [])

  const connect = useCallback(() => {
    // Don't connect if URL not resolved yet (null means not resolved)
    if (resolvedBaseUrl === null) {
      console.log('[SSE] Waiting for URL resolution...')
      return
    }
    
    // Don't connect if already connected
    if (isConnectedRef.current && eventSourceRef.current?.readyState === EventSource.OPEN) {
      return
    }
    
    // Clean up existing connection
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
    }
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }

    // Build URL with query params
    const params = new URLSearchParams()
    if (filters.length > 0) {
      params.set('filter', filters.join(','))
    }
    if (sessionId) {
      params.set('session_id', sessionId)
    }

    const url = `${resolvedBaseUrl}/api/v2/events/stream?${params.toString()}`
    console.log('[SSE] Connecting to:', url)

    try {
      const eventSource = new EventSource(url)
      eventSourceRef.current = eventSource

      eventSource.onopen = () => {
        isConnectedRef.current = true
        setConnected(true)
        setError(null)
        console.log('[SSE] Connected')
      }

      eventSource.onerror = () => {
        isConnectedRef.current = false
        setConnected(false)
        setError(new Error('SSE connection failed'))

        // Auto reconnect (only if not already scheduled)
        if (autoReconnect && !reconnectTimeoutRef.current) {
          reconnectTimeoutRef.current = setTimeout(() => {
            reconnectTimeoutRef.current = null
            console.log('[SSE] Attempting reconnect...')
            connect()
          }, reconnectDelay)
        }
      }

      // Handle all event types
      eventSource.onmessage = (e) => {
        try {
          const event: SSEEvent = JSON.parse(e.data)
          setLatestEvent(event)
          setEvents((prev) => {
            const newEvents = [...prev, event]
            // Keep only last maxEvents
            if (newEvents.length > maxEvents) {
              return newEvents.slice(-maxEvents)
            }
            return newEvents
          })
        } catch (err) {
          console.error('[SSE] Failed to parse event', err)
        }
      }

      // Also handle named events
      const eventTypes: EventType[] = [
        'session:start',
        'session:end',
        'session:update',
        'attention:warning',
        'attention:critical',
        'compact:triggered',
        'checkpoint:created',
        'port:start',
        'port:end',
        'port:blocked',
        'checklist:failed',
        'checklist:passed',
        'escalation:created',
        'escalation:resolved',
        'build:failed',
        'test:failed',
        'connection:established',
      ]

      eventTypes.forEach((type) => {
        eventSource.addEventListener(type, (e: MessageEvent) => {
          try {
            const event: SSEEvent = JSON.parse(e.data)
            setLatestEvent(event)
            setEvents((prev) => {
              const newEvents = [...prev, event]
              if (newEvents.length > maxEvents) {
                return newEvents.slice(-maxEvents)
              }
              return newEvents
            })
          } catch (err) {
            console.error('[SSE] Failed to parse event', err)
          }
        })
      })
    } catch (err) {
      setError(err instanceof Error ? err : new Error('Failed to create EventSource'))
    }
  }, [resolvedBaseUrl, filters, sessionId, maxEvents, autoReconnect, reconnectDelay])

  const reconnect = useCallback(() => {
    isConnectedRef.current = false
    connect()
  }, [connect])

  // Connect when URL is resolved
  useEffect(() => {
    if (resolvedBaseUrl !== null && !isConnectedRef.current) {
      connect()
    }

    return () => {
      isConnectedRef.current = false
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
        eventSourceRef.current = null
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
        reconnectTimeoutRef.current = null
      }
    }
  }, [resolvedBaseUrl]) // Only depend on resolvedBaseUrl, not connect

  return {
    events,
    latestEvent,
    connected,
    error,
    clearEvents,
    reconnect,
  }
}

// Helper hook for specific event types
export function useSSEEventType<T = unknown>(
  eventType: EventType,
  options: Omit<UseSSEOptions, 'filters'> = {}
) {
  const { events, ...rest } = useSSE({
    ...options,
    filters: [eventType],
  })

  const filteredEvents = events.filter((e) => e.type === eventType) as SSEEvent<T>[]
  const latestTypedEvent = filteredEvents[filteredEvents.length - 1] || null

  return {
    events: filteredEvents,
    latestEvent: latestTypedEvent,
    ...rest,
  }
}

// Convenience hooks for common event types
export function useAttentionEvents(options: Omit<UseSSEOptions, 'filters'> = {}) {
  return useSSE({
    ...options,
    filters: ['attention:warning', 'attention:critical', 'compact:triggered'],
  })
}

export function usePortEvents(options: Omit<UseSSEOptions, 'filters'> = {}) {
  return useSSE({
    ...options,
    filters: ['port:start', 'port:end', 'port:blocked'],
  })
}

export function useBuildEvents(options: Omit<UseSSEOptions, 'filters'> = {}) {
  return useSSE({
    ...options,
    filters: ['build:failed', 'test:failed', 'checklist:failed', 'checklist:passed'],
  })
}

export function useSessionEvents(options: Omit<UseSSEOptions, 'filters'> = {}) {
  return useSSE({
    ...options,
    filters: ['session:start', 'session:end', 'session:update'],
  })
}
