# Multiroom Audio MCP Server

## TL;DR

Build a Node.js/TypeScript MCP server that wraps all 24 v2 REST API endpoints from the multiroom audio system as MCP tools. The server exposes a Streamable HTTP endpoint so any AI assistant can connect to it via URL.

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

## MCP Tools to Register (24 tools from OpenAPI v2)

| Category | Tools |
|---|---|
| **Inputs** (5) | `listInputs`, `getInput`, `createInput`, `deleteInput`, `setInputEnabled` |
| **Outputs** (5) | `listOutputs`, `getOutput`, `setOutputVolume`, `setOutputMute`, `setOutputEnabled` |
| **Groups** (5) | `listGroups`, `getGroup`, `setGroupVolume`, `setGroupMute`, `setGroupEnabled` |
| **Routes** (6) | `listRoutes`, `getRoute`, `createRoute`, `deleteRoute`, `transferRoute`, `setRoutePause` |
| **Playback** (1) | `playback` |
| **Master Mute** (2) | `getMasterMute`, `setMasterMute` |

## Steps

### Phase 1 — Project scaffolding (1 file + init)

1. Initialize Node.js/TypeScript project with `package.json`
2. Install dependencies:
   - `@modelcontextprotocol/server` — MCP server SDK (v2, latest)
   - `@modelcontextprotocol/express` — Express middleware
   - `@modelcontextprotocol/node` — Node.js HTTP transport
   - `express` — HTTP framework
   - `zod` — schema validation (peer dep of MCP SDK)
   - `typescript`, `tsx` — dev tooling
3. Create `tsconfig.json`
4. Create `.gitignore`

### Phase 2 — OpenAPI-to-MCP tool definitions (2-3 files)

5. Create `src/tools.ts` — programmatically register all 24 MCP tools by reading the OpenAPI spec. Each tool:
   - Uses the `operationId` as tool name
   - Uses the `summary` + `description` as tool description
   - Defines Zod input schema from request body + path/query parameters
   - Makes HTTP calls to the multiroom REST API using the base URL passed from the CLI argument `--multiroom-url`
   - Returns the JSON response as text content

### Phase 3 — Server entry point (1 file)

6. Create `src/server.ts`:
   - Parse CLI args: `--multiroom-url <url>` (required) and `--port <port>` (default `3001`)
   - Validate that `--multiroom-url` is provided at startup; exit with usage help if missing
   - Build `McpServer` instance with `name: 'sonora-mcp'`, `version: '1.0.0'`
   - Call the tool registration function from `src/tools.ts`, passing the resolved multiroom URL
   - Set up Express app via `createMcpExpressApp()`
   - Mount MCP handler at `/mcp` endpoint
   - Listen on the configured port
   - Support CORS for local development

### Phase 4 — Convenience scripts & config

7. Add npm scripts: `dev` (tsx watch), `build` (tsc), `start` (node dist/server.js)
8. Create `README.md` with usage instructions:
   - How to run the MCP server (with required `--multiroom-url` argument)
   - Example: `node dist/server.js --multiroom-url http://localhost:8088`
   - How to configure AI assistants to connect via URL
   - CLI argument docs

## Files to Create

| File | Purpose |
|---|---|
| `package.json` | Project manifest with deps & scripts |
| `tsconfig.json` | TypeScript config |
| `.gitignore` | Ignore node_modules, dist |
| `src/server.ts` | Entry point — Express + MCP transport setup |
| `src/tools.ts` | Register all 24 tools with HTTP proxy logic |
| `src/openapi-types.ts` | TypeScript interfaces for request/response shapes (optional, for type safety) |
| `README.md` | Usage docs |

## Key Design Decisions

1. **Transport**: Streamable HTTP (not stdio) — user needs URL-based connection to AI assistants
2. **Stateless mode**: `sessionIdGenerator: undefined` — simple, no session management needed since multiroom API is stateless
3. **CLI arguments** (no env vars, no defaults for required args):
   - `--multiroom-url <url>` (**required**) — base URL for multiroom REST API (e.g. `http://localhost:8088`)
   - `--port <number>` (optional, default `3001`) — port for the MCP server
4. **Error handling**: Multiroom API errors (RFC 7807) are passed through as tool error results
5. **No authentication**: Multiroom API has no auth; MCP server will also be open (local use)

## Verification

1. Run `npm run build` — TypeScript compiles without errors
2. Run `npm run dev -- --multiroom-url http://localhost:8088` — server starts on port 3001 and connects to multiroom
3. Use MCP Inspector (`npx @modelcontextprotocol/inspector`) to connect to `http://localhost:3001/mcp` and verify all 24 tools are listed
4. Call a read-only tool like `listOutputs` to verify proxy to multiroom works

## Scope & Decisions

- **Scope**: v2 API only (not legacy v1)
- **Out of scope**: OAuth/auth, resources, prompts (tools only for now)
- **Framework**: Express with `@modelcontextprotocol/express` middleware (most familiar to users, cleanest setup)
- **Language**: TypeScript (type safety for tool schemas)

## Open Questions

1. **Multiroom API URL**: The `--multiroom-url` is required at startup. Example: `http://localhost:8088`. The server refuses to start without it.
2. **Tool granularity**: One tool per REST endpoint (24 tools). Would you prefer fewer, higher-level tools (e.g., a single `manageAudio` tool with sub-commands)?
3. **Streaming/notifications**: The MCP SDK supports SSE notifications. Would you want the server to push status updates (e.g., route status changes) to connected clients?
