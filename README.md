# sonora-mcp

MCP server that wraps the [Multiroom Audio Hub v2 REST API](openapi.json) as 24 MCP tools. Any AI assistant that supports MCP can connect to this server via a URL and control your multiroom audio system.

## Architecture

```
AI Assistant (Copilot, Claude, etc.)
       │
       ▼  (MCP protocol over HTTP)
┌──────────────────────┐
│  sonora-mcp server   │  port 3001
│  (Node.js/TypeScript)│
└──────────┬───────────┘
           │  HTTP REST calls
           ▼
┌──────────────────────┐
│  multiroom-ai REST   │  port 8088
│  (Spring Boot)       │
└──────────────────────┘
```

## Quick Start

### Build

```bash
npm install
npm run build
```

### Run

```bash
# Start the MCP server, pointing at your Multiroom API
node dist/server.js --multiroom-url http://localhost:8088
```

### Development

```bash
npm run dev -- --multiroom-url http://localhost:8088
```

## CLI Arguments

| Argument | Required | Default | Description |
|---|---|---|---|
| `--multiroom-url <url>` | **Yes** | — | Base URL of the Multiroom Audio REST API (e.g. `http://localhost:8088`) |
| `--port <port>` | No | `3001` | Port for the MCP server |
| `-h, --help` | No | — | Show usage help |

The server **will not start** without `--multiroom-url`. If you omit it, you'll see a usage message and exit.

## Connecting AI Assistants

### MCP Inspector (testing)

```bash
npx @modelcontextprotocol/inspector
```

Connect to: `http://localhost:3001/mcp`

### GitHub Copilot (VS Code)

Add to your VS Code `settings.json`:

```json
{
  "mcp": {
    "servers": {
      "sonora-mcp": {
        "type": "http",
        "url": "http://localhost:3001/mcp"
      }
    }
  }
}
```

### Claude Desktop

Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "sonora-mcp": {
      "type": "streamableHttp",
      "url": "http://localhost:3001/mcp"
    }
  }
}
```

## Available Tools (24)

### Inputs (5)

| Tool | Description |
|---|---|
| `listInputs` | List all audio inputs |
| `getInput` | Get a single input by ID |
| `createInput` | Create a new ephemeral audio input |
| `deleteInput` | Delete an ephemeral audio input |
| `setInputEnabled` | Enable or disable an audio input |

### Outputs (5)

| Tool | Description |
|---|---|
| `listOutputs` | List all audio outputs |
| `getOutput` | Get a single output by ID |
| `setOutputVolume` | Set output volume (0–100) |
| `setOutputMute` | Mute or unmute an output |
| `setOutputEnabled` | Enable or disable an output |

### Groups (5)

| Tool | Description |
|---|---|
| `listGroups` | List all audio output groups |
| `getGroup` | Get a single group by ID |
| `setGroupVolume` | Set volume for all outputs in a group |
| `setGroupMute` | Mute or unmute all outputs in a group |
| `setGroupEnabled` | Enable or disable a group |

### Routes (6)

| Tool | Description |
|---|---|
| `listRoutes` | List all active audio routes |
| `getRoute` | Get a single route by ID |
| `createRoute` | Create a new audio route (input → output/group) |
| `deleteRoute` | Stop and delete a route |
| `transferRoute` | Transfer a route to a new output/group |
| `setRoutePause` | Pause or resume a route |

### Playback (1)

| Tool | Description |
|---|---|
| `playback` | Play audio from a URI (auto-creates input + route) |

### Master Mute (2)

| Tool | Description |
|---|---|
| `getMasterMute` | Get system-wide master mute state |
| `setMasterMute` | Set system-wide master mute |

## Health Check

```bash
curl http://localhost:3001/health
# → {"status":"ok","server":"sonora-mcp","version":"1.0.0"}
```

## Tech Stack

- **Runtime**: Node.js 18+
- **Language**: TypeScript
- **MCP SDK**: `@modelcontextprotocol/sdk` (v1.30+)
- **Transport**: Streamable HTTP (stateless)
- **HTTP**: Express 5
- **Schema validation**: Zod

## License

AGPL-3.0
