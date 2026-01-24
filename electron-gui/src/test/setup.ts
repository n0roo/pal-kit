import '@testing-library/jest-dom'

// Mock window.pal API
const mockPal = {
  request: vi.fn().mockResolvedValue({ data: null, error: null }),
  getStatus: vi.fn().mockResolvedValue({ data: null, error: null }),
  getOrchestrations: vi.fn().mockResolvedValue({ data: [], error: null }),
  getSessions: vi.fn().mockResolvedValue({ data: [], error: null }),
  getSessionHierarchy: vi.fn().mockResolvedValue({ data: null, error: null }),
  getAgents: vi.fn().mockResolvedValue({ data: [], error: null }),
  getAgentVersions: vi.fn().mockResolvedValue({ data: [], error: null }),
  getAttention: vi.fn().mockResolvedValue({ data: null, error: null }),
  getAttentionHistory: vi.fn().mockResolvedValue({ data: [], error: null }),
}

// Mock window.app API
const mockApp = {
  getVersion: vi.fn().mockResolvedValue('1.0.0'),
  getPlatform: vi.fn().mockResolvedValue('darwin'),
  getServerPort: vi.fn().mockResolvedValue(9000),
  isServerRunning: vi.fn().mockResolvedValue(true),
  restartServer: vi.fn().mockResolvedValue(true),
}

// Assign to window
Object.defineProperty(window, 'pal', { value: mockPal, writable: true })
Object.defineProperty(window, 'app', { value: mockApp, writable: true })

// Reset mocks before each test
beforeEach(() => {
  vi.clearAllMocks()
})
