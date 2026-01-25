import React from 'react'
import { useSSE, SSEEvent, EventType } from '../hooks/useSSE'

const EVENT_ICONS: Record<string, string> = {
  'session:start': '🚀',
  'session:end': '🏁',
  'session:update': '🔄',
  'attention:warning': '⚠️',
  'attention:critical': '🚨',
  'compact:triggered': '📦',
  'checkpoint:created': '💾',
  'port:start': '▶️',
  'port:end': '✅',
  'port:blocked': '🚫',
  'checklist:failed': '❌',
  'checklist:passed': '✔️',
  'escalation:created': '📢',
  'escalation:resolved': '👍',
  'build:failed': '🔴',
  'test:failed': '🧪',
  'connection:established': '🔗',
}

const EVENT_COLORS: Record<string, string> = {
  'session:start': 'bg-green-100 border-green-300',
  'session:end': 'bg-gray-100 border-gray-300',
  'attention:warning': 'bg-yellow-100 border-yellow-300',
  'attention:critical': 'bg-red-100 border-red-300',
  'compact:triggered': 'bg-orange-100 border-orange-300',
  'port:start': 'bg-blue-100 border-blue-300',
  'port:end': 'bg-green-100 border-green-300',
  'checklist:failed': 'bg-red-100 border-red-300',
  'checklist:passed': 'bg-green-100 border-green-300',
  'build:failed': 'bg-red-100 border-red-300',
  'test:failed': 'bg-red-100 border-red-300',
}

interface EventItemProps {
  event: SSEEvent
}

function EventItem({ event }: EventItemProps) {
  const icon = EVENT_ICONS[event.type] || '📌'
  const colorClass = EVENT_COLORS[event.type] || 'bg-gray-50 border-gray-200'
  const time = new Date(event.timestamp).toLocaleTimeString()

  return (
    <div className={`p-3 mb-2 rounded-lg border ${colorClass}`}>
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="text-lg">{icon}</span>
          <span className="font-medium text-sm">{event.type}</span>
        </div>
        <span className="text-xs text-gray-500">{time}</span>
      </div>
      {event.session_id && (
        <div className="text-xs text-gray-600 mt-1">
          Session: {event.session_id}
        </div>
      )}
      {event.port_id && (
        <div className="text-xs text-gray-600">Port: {event.port_id}</div>
      )}
      {event.data && (
        <div className="mt-2 text-xs bg-white bg-opacity-50 p-2 rounded overflow-auto max-h-24">
          <pre>{JSON.stringify(event.data, null, 2)}</pre>
        </div>
      )}
    </div>
  )
}

interface EventFeedProps {
  filters?: EventType[]
  maxEvents?: number
  title?: string
}

export function EventFeed({
  filters,
  maxEvents = 50,
  title = 'Real-time Events',
}: EventFeedProps) {
  const { events, connected, error, clearEvents, reconnect } = useSSE({
    filters,
    maxEvents,
    autoReconnect: true,
  })

  return (
    <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
      {/* Header */}
      <div className="px-4 py-3 bg-gray-50 border-b border-gray-200 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h3 className="font-semibold text-gray-800">{title}</h3>
          <div
            className={`w-2 h-2 rounded-full ${
              connected ? 'bg-green-500' : 'bg-red-500'
            }`}
            title={connected ? 'Connected' : 'Disconnected'}
          />
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs text-gray-500">{events.length} events</span>
          <button
            onClick={clearEvents}
            className="text-xs text-gray-500 hover:text-gray-700"
            title="Clear events"
          >
            Clear
          </button>
          {!connected && (
            <button
              onClick={reconnect}
              className="text-xs text-blue-500 hover:text-blue-700"
            >
              Reconnect
            </button>
          )}
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="px-4 py-2 bg-red-50 text-red-600 text-sm">
          {error.message}
        </div>
      )}

      {/* Events list */}
      <div className="p-4 max-h-96 overflow-y-auto">
        {events.length === 0 ? (
          <div className="text-center text-gray-400 py-8">
            Waiting for events...
          </div>
        ) : (
          events
            .slice()
            .reverse()
            .map((event, index) => <EventItem key={index} event={event} />)
        )}
      </div>
    </div>
  )
}

// Compact notification banner component
export function AttentionBanner() {
  const { latestEvent, connected } = useSSE({
    filters: ['attention:warning', 'attention:critical', 'compact:triggered'],
    maxEvents: 1,
  })

  if (!connected || !latestEvent) return null

  const isWarning = latestEvent.type === 'attention:warning'
  const isCritical =
    latestEvent.type === 'attention:critical' ||
    latestEvent.type === 'compact:triggered'

  if (!isWarning && !isCritical) return null

  const data = latestEvent.data as { usage_percent?: number; warning?: string }

  return (
    <div
      className={`fixed top-0 left-0 right-0 z-50 px-4 py-2 text-center text-sm ${
        isCritical
          ? 'bg-red-500 text-white'
          : 'bg-yellow-400 text-yellow-900'
      }`}
    >
      {isCritical ? '🚨' : '⚠️'}{' '}
      {data?.warning || `Token usage: ${data?.usage_percent?.toFixed(1)}%`}
    </div>
  )
}

// Build status indicator
export function BuildStatusIndicator() {
  const { latestEvent } = useSSE({
    filters: ['build:failed', 'test:failed', 'checklist:passed'],
    maxEvents: 1,
  })

  if (!latestEvent) return null

  const isFailed =
    latestEvent.type === 'build:failed' || latestEvent.type === 'test:failed'
  const isPassed = latestEvent.type === 'checklist:passed'

  return (
    <div
      className={`flex items-center gap-1 px-2 py-1 rounded text-xs ${
        isFailed
          ? 'bg-red-100 text-red-700'
          : isPassed
          ? 'bg-green-100 text-green-700'
          : 'bg-gray-100 text-gray-600'
      }`}
    >
      {isFailed ? '🔴' : isPassed ? '🟢' : '⚪'}
      <span>
        {isFailed
          ? 'Build Failed'
          : isPassed
          ? 'Build Passed'
          : 'Unknown'}
      </span>
    </div>
  )
}

export default EventFeed
