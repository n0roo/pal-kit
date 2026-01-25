import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { useApi, useOrchestrations, useAgents, useApp } from '../hooks/useApi'

describe('useApi', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('should fetch status on mount', async () => {
    const mockStatus = {
      orchestrations: { total: 5, running: 2 },
      builds: { total: 10, active: 3 },
      agents: { total: 4 },
    }
    // useApi uses window.pal.request() internally
    window.pal.request = vi.fn().mockResolvedValue({ data: mockStatus, error: null })

    const { result } = renderHook(() => useApi())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.status).toEqual(mockStatus)
    expect(result.current.error).toBeNull()
    expect(window.pal.request).toHaveBeenCalledWith('/status', undefined)
  })

  it('should handle error', async () => {
    window.pal.request = vi.fn().mockResolvedValue({ data: null, error: 'Connection failed' })

    const { result } = renderHook(() => useApi())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.status).toBeNull()
    expect(result.current.error).toBe('Connection failed')
  })
})

describe('useOrchestrations', () => {
  it('should fetch orchestrations', async () => {
    const mockOrchestrations = [
      { id: '1', title: 'Orch 1', status: 'running', progress_percent: 50 },
      { id: '2', title: 'Orch 2', status: 'complete', progress_percent: 100 },
    ]
    window.pal.request = vi.fn().mockResolvedValue({ data: mockOrchestrations, error: null })

    const { result } = renderHook(() => useOrchestrations())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.orchestrations).toEqual(mockOrchestrations)
    expect(result.current.orchestrations).toHaveLength(2)
  })

  it('should return empty array on error', async () => {
    window.pal.request = vi.fn().mockResolvedValue({ data: null, error: 'Error' })

    const { result } = renderHook(() => useOrchestrations())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.orchestrations).toEqual([])
  })
})

describe('useAgents', () => {
  it('should fetch agents', async () => {
    const mockAgents = [
      { id: 'agent-1', name: 'Worker 1', type: 'worker' },
      { id: 'agent-2', name: 'Operator 1', type: 'operator' },
    ]
    window.pal.request = vi.fn().mockResolvedValue({ data: mockAgents, error: null })

    const { result } = renderHook(() => useAgents())

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.agents).toEqual(mockAgents)
  })
})

describe('useApp', () => {
  it('should check server status', async () => {
    window.app.isServerRunning = vi.fn().mockResolvedValue(true)
    window.app.getServerPort = vi.fn().mockResolvedValue(9000)

    const { result } = renderHook(() => useApp())

    await waitFor(() => {
      expect(result.current.serverRunning).toBe(true)
    })

    expect(result.current.serverPort).toBe(9000)
  })

  it('should restart server', async () => {
    window.app.restartServer = vi.fn().mockResolvedValue(true)
    window.app.isServerRunning = vi.fn().mockResolvedValue(true)
    window.app.getServerPort = vi.fn().mockResolvedValue(9001)

    const { result } = renderHook(() => useApp())

    const success = await result.current.restartServer()

    expect(success).toBe(true)
    expect(window.app.restartServer).toHaveBeenCalled()
  })
})
