# Taskwondo MCPB — Claude Desktop Extension

One-click MCP server installation for Claude Desktop on Windows.

## Building

From the repository root:

```
make build-mcpb
```

This cross-compiles the MCP server for Windows and packages it into `build/mcpb/taskwondo.mcpb`.

## Installing on Claude Desktop (Windows)

### Option A: Extension Installer (recommended)

1. Open **Claude Desktop**
2. Go to **Settings** (gear icon) → **Extensions** → **Extension Developer**
3. Click **Install Extension…** and select the `taskwondo.mcpb` file
4. Fill in the **Server URL** (e.g. `https://taskwondo.example.com`)
5. Optionally fill in your **API Key**, or leave it blank and use the `login` tool later for browser-based auth
6. Click **Install**

### Option B: Drag and drop

1. Open **Claude Desktop**
2. Drag the `taskwondo.mcpb` file into the Claude Desktop window
3. Follow the prompts to configure and install

### Option C: Manual configuration

If the extension installer is not available, you can configure the MCP server manually:

1. Build the Windows binary:
   ```
   make build-mcp-windows
   ```

2. Copy `build/mcp/taskwondo-mcp.exe` to a permanent location on your Windows machine, e.g.:
   ```
   C:\Users\<you>\AppData\Local\taskwondo\taskwondo-mcp.exe
   ```

3. Open the Claude Desktop config file at:
   ```
   %APPDATA%\Claude\claude_desktop_config.json
   ```
   (Press `Win+R`, paste the path above, and press Enter to open the folder)

4. Add the Taskwondo server to `mcpServers`:
   ```json
   {
     "mcpServers": {
       "taskwondo": {
         "command": "C:\\Users\\<you>\\AppData\\Local\\taskwondo\\taskwondo-mcp.exe",
         "env": {
           "TASKWONDO_URL": "https://taskwondo.example.com"
         }
       }
     }
   }
   ```
   You can optionally add `"TASKWONDO_API_KEY": "twk_..."` to the `env` block, or skip it and use the `login` tool after starting Claude Desktop.

5. Restart Claude Desktop

## Authentication

Two options:

- **API Key**: Provide your API key during installation (or in the config). Find it in the Taskwondo web UI under **User menu → API Keys**.
- **Browser Login**: Leave the API key blank. In Claude Desktop, ask Claude to use the `login` tool. A browser window will open for you to authorize, and the API key is saved automatically.
