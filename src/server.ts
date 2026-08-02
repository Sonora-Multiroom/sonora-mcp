import { createMcpExpressApp } from "@modelcontextprotocol/express";
import { NodeStreamableHTTPServerTransport } from "@modelcontextprotocol/node";
import { McpServer } from "@modelcontextprotocol/server";
import express from "express";
import cors from "cors";

import { registerTools } from "./tools.js";

// ─── CLI argument parsing ───────────────────────────────────────────────────

function parseArgs(argv: string[]): { multiroomUrl: string; port: number } {
  let multiroomUrl: string | undefined;
  let port = 3001;

  for (let i = 2; i < argv.length; i++) {
    switch (argv[i]) {
      case "--multiroom-url":
        multiroomUrl = argv[++i];
        break;
      case "--port":
        port = parseInt(argv[++i], 10);
        break;
      case "--help":
      case "-h":
        printUsage();
        process.exit(0);
    }
  }

  if (!multiroomUrl) {
    console.error("Error: --multiroom-url is required\n");
    printUsage();
    process.exit(1);
  }

  // Strip trailing slash
  multiroomUrl = multiroomUrl.replace(/\/+$/, "");

  return { multiroomUrl, port };
}

function printUsage(): void {
  console.log(`Usage: sonora-mcp --multiroom-url <url> [--port <port>]

Options:
  --multiroom-url <url>   (required) Base URL for the Multiroom Audio REST API
                          e.g. http://localhost:8088
  --port <port>           (optional) Port for this MCP server (default: 3001)
  -h, --help              Show this help message`);
}

// ─── Main ───────────────────────────────────────────────────────────────────

async function main(): Promise<void> {
  const { multiroomUrl, port } = parseArgs(process.argv);

  console.log(`sonora-mcp v1.0.0`);
  console.log(`Multiroom API: ${multiroomUrl}`);
  console.log(`MCP server starting on port ${port}...`);

  // Build MCP server
  const mcpServer = new McpServer({
    name: "sonora-mcp",
    version: "1.0.0",
  });

  // Register all 24 tools
  registerTools(mcpServer, multiroomUrl);

  // Express app with CORS for local dev
  const app = express();
  app.use(cors());
  app.use(express.json());

  // Request logging middleware
  app.use((req, _res, next) => {
    const timestamp = new Date().toISOString();
    console.log(`[${timestamp}] ${req.method} ${req.originalUrl} from ${req.ip}`);
    next();
  });

  // Mount MCP handler at /mcp
  app.all("/mcp", async (req, res) => {
    const timestamp = new Date().toISOString();

    // Log the MCP JSON-RPC request body if present
    const body = req.body as Record<string, unknown> | undefined;
    if (body) {
      const method = body.method as string | undefined;
      const params = body.params as Record<string, unknown> | undefined;

      if (method === "tools/call") {
        const toolName = params?.name as string | undefined;
        const toolArgs = params?.arguments as Record<string, unknown> | undefined;
        console.log(`[${timestamp}] → MCP tools/call: ${toolName}`);
        if (toolArgs) {
          console.log(`[${timestamp}]   args: ${JSON.stringify(toolArgs)}`);
        }
      } else if (method) {
        console.log(`[${timestamp}] → MCP ${method}`);
      } else {
        console.log(`[${timestamp}] → MCP request (no method)`);
      }
    } else {
      console.log(`[${timestamp}] → MCP ${req.method} (no body)`);
    }

    const transport = new NodeStreamableHTTPServerTransport({
      sessionIdGenerator: undefined, // stateless
    });
    await mcpServer.connect(transport);
    await transport.handleRequest(req, res, req.body);

    const endTimestamp = new Date().toISOString();
    console.log(`[${endTimestamp}] ← MCP response sent (${res.statusCode})`);
  });

  // Health check
  app.get("/health", (_req, res) => {
    res.json({ status: "ok", server: "sonora-mcp", version: "1.0.0" });
  });

  app.listen(port, () => {
    console.log(`sonora-mcp listening on http://localhost:${port}/mcp`);
    console.log(`Health check: http://localhost:${port}/health`);
  });
}

main().catch((err) => {
  console.error("Fatal error:", err);
  process.exit(1);
});
