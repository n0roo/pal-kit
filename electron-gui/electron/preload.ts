import { contextBridge, ipcRenderer } from 'electron'

// API Response type
interface ApiResponse<T = any> {
  data: T | null
  error: string | null
}

// Expose PAL API via IPC
contextBridge.exposeInMainWorld('pal', {
  // Generic request
  request: (endpoint: string, options?: { method?: string; body?: any }): Promise<ApiResponse> =>
    ipcRenderer.invoke('pal:request', endpoint, options),

  // Status
  getStatus: (): Promise<ApiResponse> => 
    ipcRenderer.invoke('pal:status'),

  // Orchestrations
  getOrchestrations: (status?: string): Promise<ApiResponse> => 
    ipcRenderer.invoke('pal:orchestrations', status),

  // Sessions
  getSessions: (): Promise<ApiResponse> => 
    ipcRenderer.invoke('pal:sessions'),
  
  getSessionHierarchy: (id: string): Promise<ApiResponse> => 
    ipcRenderer.invoke('pal:session-hierarchy', id),

  // Agents
  getAgents: (type?: string): Promise<ApiResponse> => 
    ipcRenderer.invoke('pal:agents', type),
  
  getAgentVersions: (agentId: string): Promise<ApiResponse> => 
    ipcRenderer.invoke('pal:agent-versions', agentId),

  // Attention
  getAttention: (sessionId: string): Promise<ApiResponse> => 
    ipcRenderer.invoke('pal:attention', sessionId),
  
  getAttentionHistory: (sessionId: string, limit?: number): Promise<ApiResponse> => 
    ipcRenderer.invoke('pal:attention-history', sessionId, limit),
})

// App utilities
contextBridge.exposeInMainWorld('app', {
  getVersion: (): Promise<string> => ipcRenderer.invoke('app:version'),
  getPlatform: (): Promise<string> => ipcRenderer.invoke('app:platform'),
  getServerPort: (): Promise<number> => ipcRenderer.invoke('app:server-port'),
  isServerRunning: (): Promise<boolean> => ipcRenderer.invoke('app:server-running'),
  restartServer: (): Promise<boolean> => ipcRenderer.invoke('app:restart-server'),
  // File explorer
  selectFolder: (): Promise<{ path: string | null; canceled: boolean }> =>
    ipcRenderer.invoke('app:select-folder'),
  openInExplorer: (folderPath: string): Promise<{ success: boolean; error?: string }> =>
    ipcRenderer.invoke('app:open-in-explorer', folderPath),
  showItemInFolder: (itemPath: string): Promise<{ success: boolean }> =>
    ipcRenderer.invoke('app:show-item-in-folder', itemPath),
})

// Type declarations
declare global {
  interface Window {
    pal: {
      request: (endpoint: string, options?: { method?: string; body?: any }) => Promise<ApiResponse>
      getStatus: () => Promise<ApiResponse>
      getOrchestrations: (status?: string) => Promise<ApiResponse>
      getSessions: () => Promise<ApiResponse>
      getSessionHierarchy: (id: string) => Promise<ApiResponse>
      getAgents: (type?: string) => Promise<ApiResponse>
      getAgentVersions: (agentId: string) => Promise<ApiResponse>
      getAttention: (sessionId: string) => Promise<ApiResponse>
      getAttentionHistory: (sessionId: string, limit?: number) => Promise<ApiResponse>
    }
    app: {
      getVersion: () => Promise<string>
      getPlatform: () => Promise<string>
      getServerPort: () => Promise<number>
      isServerRunning: () => Promise<boolean>
      restartServer: () => Promise<boolean>
      selectFolder: () => Promise<{ path: string | null; canceled: boolean }>
      openInExplorer: (folderPath: string) => Promise<{ success: boolean; error?: string }>
      showItemInFolder: (itemPath: string) => Promise<{ success: boolean }>
    }
  }
}

export {}
