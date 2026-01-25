import { app, BrowserWindow, ipcMain, shell, dialog } from 'electron'
import { spawn, ChildProcess, execSync } from 'child_process'
import path from 'path'
import fs from 'fs'
import net from 'net'
import http from 'http'

let mainWindow: BrowserWindow | null = null
let serverProcess: ChildProcess | null = null
let serverPort: number = 0
let serverStarting = false

// ============================================
// Dynamic Port Allocation
// ============================================

function isPortAvailable(port: number): Promise<boolean> {
  return new Promise((resolve) => {
    const server = net.createServer()
    server.once('error', () => resolve(false))
    server.once('listening', () => {
      server.close(() => resolve(true))
    })
    server.listen(port, '127.0.0.1')
  })
}

async function findAvailablePort(startPort = 9000, endPort = 9999): Promise<number> {
  for (let port = startPort; port <= endPort; port++) {
    if (await isPortAvailable(port)) {
      return port
    }
  }
  throw new Error('No available port found')
}

// ============================================
// PAL Server Management
// ============================================

function findPalBinary(): string | null {
  const projectRoot = path.resolve(__dirname, '../..')
  const isWindows = process.platform === 'win32'
  const binaryName = isWindows ? 'pal.exe' : 'pal'

  const possiblePaths = [
    // Development paths
    path.join(projectRoot, binaryName),
    path.join(projectRoot, 'pal'),
    // Production paths (packaged app)
    path.join(process.resourcesPath || '', binaryName),
    path.join(process.resourcesPath || '', 'pal'),
  ]

  for (const p of possiblePaths) {
    try {
      if (fs.existsSync(p)) {
        if (!isWindows) {
          fs.accessSync(p, fs.constants.X_OK)
        }
        console.log(`[Main] Found pal binary: ${p}`)
        return p
      }
    } catch { continue }
  }

  // Fallback: check PATH
  try {
    const cmd = isWindows ? 'where pal' : 'which pal'
    return execSync(cmd, { encoding: 'utf-8' }).trim().split('\n')[0]
  } catch { return null }
}

async function startServer(): Promise<number> {
  // Prevent concurrent starts
  if (serverStarting) {
    console.log('[Main] Server already starting, waiting...')
    // Wait for current start to finish
    for (let i = 0; i < 30; i++) {
      await new Promise(r => setTimeout(r, 500))
      if (!serverStarting && serverPort > 0) {
        return serverPort
      }
    }
    return 0
  }

  // If server already running, return current port
  if (serverProcess && serverPort > 0) {
    console.log(`[Main] Server already running on port ${serverPort}`)
    return serverPort
  }

  serverStarting = true

  try {
    const palBinary = findPalBinary()
    if (!palBinary) {
      console.error('[Main] PAL binary not found!')
      return 0
    }

    // Find available port
    try {
      serverPort = await findAvailablePort()
    } catch (err) {
      console.error('[Main] No available port:', err)
      return 0
    }

    const projectRoot = path.resolve(palBinary, '..')
    console.log(`[Main] Starting PAL server on port ${serverPort}`)

    return new Promise((resolve) => {
      serverProcess = spawn(palBinary, ['serve', '--port', String(serverPort)], {
        cwd: projectRoot,
        stdio: ['ignore', 'pipe', 'pipe'],
      })

      serverProcess.stdout?.on('data', (d) => {
        const msg = d.toString().trim()
        console.log(`[PAL] ${msg}`)
        // Detect server ready
        if (msg.includes('running at')) {
          console.log(`[Main] Server ready on port ${serverPort}`)
          serverStarting = false
          resolve(serverPort)
        }
      })

      serverProcess.stderr?.on('data', (d) => {
        const msg = d.toString().trim()
        console.error(`[PAL] ${msg}`)
        // Port conflict - try next port
        if (msg.includes('address already in use')) {
          console.log('[Main] Port conflict, trying next port...')
          serverProcess?.kill()
          serverProcess = null
          serverPort = 0
          serverStarting = false
          // Retry with next port
          startServer().then(resolve)
        }
      })
      
      serverProcess.on('error', (err) => {
        console.error('[Main] Server spawn error:', err)
        serverStarting = false
        resolve(0)
      })

      serverProcess.on('exit', (code) => {
        console.log(`[Main] Server exited with code ${code}`)
        serverProcess = null
        // Don't reset port here - let the retry logic handle it
      })

      // Timeout fallback
      setTimeout(() => {
        if (serverStarting) {
          console.log('[Main] Server startup timeout, checking if running...')
          serverStarting = false
          // Check if server is actually responding
          apiRequest('/status')
            .then(() => resolve(serverPort))
            .catch(() => resolve(0))
        }
      }, 15000)
    })
  } finally {
    // Ensure flag is reset even on error
    setTimeout(() => { serverStarting = false }, 20000)
  }
}

function stopServer() {
  if (serverProcess) {
    console.log('[Main] Stopping server...')
    serverProcess.kill('SIGTERM')
    serverProcess = null
  }
  serverPort = 0
  serverStarting = false
}

// ============================================
// Internal API Request (No CORS)
// ============================================

function apiRequest(endpoint: string, options: { method?: string; body?: any } = {}): Promise<any> {
  return new Promise((resolve, reject) => {
    if (!serverPort) {
      reject(new Error('Server not running'))
      return
    }

    const method = options.method || 'GET'
    
    const req = http.request({
      hostname: '127.0.0.1',
      port: serverPort,
      path: `/api/v2${endpoint}`,
      method,
      headers: options.body ? { 'Content-Type': 'application/json' } : {},
    }, (res) => {
      let data = ''
      res.on('data', chunk => data += chunk)
      res.on('end', () => {
        try {
          resolve(JSON.parse(data))
        } catch {
          resolve(data || null)
        }
      })
    })

    req.on('error', (err) => {
      reject(err)
    })

    req.setTimeout(10000, () => {
      req.destroy()
      reject(new Error('Request timeout'))
    })

    if (options.body) {
      req.write(JSON.stringify(options.body))
    }
    req.end()
  })
}

// ============================================
// IPC Handlers - API Proxy
// ============================================

ipcMain.handle('pal:request', async (_, endpoint: string, options?: { method?: string; body?: any }) => {
  try {
    return { data: await apiRequest(endpoint, options || {}), error: null }
  } catch (err: any) {
    return { data: null, error: err.message }
  }
})

ipcMain.handle('pal:status', async () => {
  try {
    return { data: await apiRequest('/status'), error: null }
  } catch (err: any) {
    return { data: null, error: err.message }
  }
})

ipcMain.handle('pal:orchestrations', async (_, status?: string) => {
  try {
    const query = status ? `?status=${status}` : ''
    return { data: await apiRequest(`/orchestrations${query}`), error: null }
  } catch (err: any) {
    return { data: null, error: err.message }
  }
})

ipcMain.handle('pal:sessions', async () => {
  try {
    return { data: await apiRequest('/sessions/builds?limit=20'), error: null }
  } catch (err: any) {
    return { data: null, error: err.message }
  }
})

ipcMain.handle('pal:session-hierarchy', async (_, id: string) => {
  try {
    return { data: await apiRequest(`/sessions/hierarchy/${id}/tree`), error: null }
  } catch (err: any) {
    return { data: null, error: err.message }
  }
})

ipcMain.handle('pal:agents', async (_, type?: string) => {
  try {
    const query = type ? `?type=${type}` : ''
    return { data: await apiRequest(`/agents${query}`), error: null }
  } catch (err: any) {
    return { data: null, error: err.message }
  }
})

ipcMain.handle('pal:agent-versions', async (_, agentId: string) => {
  try {
    return { data: await apiRequest(`/agents/${agentId}/versions`), error: null }
  } catch (err: any) {
    return { data: null, error: err.message }
  }
})

ipcMain.handle('pal:attention', async (_, sessionId: string) => {
  try {
    return { data: await apiRequest(`/attention/${sessionId}`), error: null }
  } catch (err: any) {
    return { data: null, error: err.message }
  }
})

ipcMain.handle('pal:attention-history', async (_, sessionId: string, limit = 10) => {
  try {
    return { data: await apiRequest(`/attention/${sessionId}/history?limit=${limit}`), error: null }
  } catch (err: any) {
    return { data: null, error: err.message }
  }
})

// App info
ipcMain.handle('app:version', () => app.getVersion())
ipcMain.handle('app:platform', () => process.platform)
ipcMain.handle('app:server-port', () => serverPort)
ipcMain.handle('app:server-running', () => serverProcess !== null && serverPort > 0)

ipcMain.handle('app:restart-server', async () => {
  stopServer()
  await new Promise(r => setTimeout(r, 1000))
  const port = await startServer()
  return port > 0
})

// File explorer handlers
ipcMain.handle('app:select-folder', async () => {
  if (!mainWindow) return { path: null, canceled: true }

  const result = await dialog.showOpenDialog(mainWindow, {
    properties: ['openDirectory', 'createDirectory'],
    title: '프로젝트 폴더 선택',
    buttonLabel: '선택',
  })

  return {
    path: result.filePaths[0] || null,
    canceled: result.canceled,
  }
})

ipcMain.handle('app:open-in-explorer', async (_, folderPath: string) => {
  try {
    await shell.openPath(folderPath)
    return { success: true }
  } catch (err: any) {
    return { success: false, error: err.message }
  }
})

ipcMain.handle('app:show-item-in-folder', async (_, itemPath: string) => {
  shell.showItemInFolder(itemPath)
  return { success: true }
})

// ============================================
// Window Management
// ============================================

const createWindow = async () => {
  // Start server first (with retry)
  let port = await startServer()
  if (!port) {
    console.log('[Main] First start failed, retrying...')
    await new Promise(r => setTimeout(r, 2000))
    port = await startServer()
  }
  console.log(`[Main] Final server port: ${port}`)

  mainWindow = new BrowserWindow({
    width: 1400,
    height: 900,
    minWidth: 1024,
    minHeight: 768,
    webPreferences: {
      preload: path.join(__dirname, 'preload.js'),
      nodeIntegration: false,
      contextIsolation: true,
    },
    titleBarStyle: 'hiddenInset',
    backgroundColor: '#0f172a',
    show: false,
  })

  mainWindow.once('ready-to-show', () => mainWindow?.show())

  if (process.env.VITE_DEV_SERVER_URL) {
    mainWindow.loadURL(process.env.VITE_DEV_SERVER_URL)
    mainWindow.webContents.openDevTools()
  } else {
    mainWindow.loadFile(path.join(__dirname, '../dist/index.html'))
  }

  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url)
    return { action: 'deny' }
  })

  mainWindow.on('closed', () => { mainWindow = null })
}

// ============================================
// App Lifecycle
// ============================================

// Ensure single instance
const gotTheLock = app.requestSingleInstanceLock()

if (!gotTheLock) {
  console.log('[Main] Another instance is running, quitting...')
  app.quit()
} else {
  app.on('second-instance', () => {
    if (mainWindow) {
      if (mainWindow.isMinimized()) mainWindow.restore()
      mainWindow.focus()
    }
  })

  app.whenReady().then(createWindow)

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) createWindow()
  })

  app.on('window-all-closed', () => {
    stopServer()
    if (process.platform !== 'darwin') app.quit()
  })

  app.on('before-quit', stopServer)

  process.on('uncaughtException', (err) => {
    console.error('[Main] Uncaught exception:', err)
    stopServer()
  })

  process.on('SIGTERM', () => {
    stopServer()
    app.quit()
  })

  process.on('SIGINT', () => {
    stopServer()
    app.quit()
  })
}
