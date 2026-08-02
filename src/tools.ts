import { McpServer, type CallToolResult } from "@modelcontextprotocol/server";
import { z } from "zod";

// ZodRawShape type from the MCP SDK for registerTool compatibility
type ZodRawShape = Record<string, z.ZodTypeAny>;

// ─── OpenAPI schema types (subset used for tool definitions) ────────────────

interface ToolDef {
  method: string;
  path: string;
  operationId: string;
  summary: string;
  description: string;
  parameters: ParameterDef[];
  requestBody?: RequestBodyDef;
  pathTemplate: string;
}

interface ParameterDef {
  name: string;
  in: "path" | "query";
  description: string;
  required: boolean;
  schema: { type: string; enum?: string[] };
}

interface RequestBodyDef {
  required: boolean;
  contentType: string;
  schemaRef: string;
}

// ─── Tool definitions derived from the OpenAPI v2 spec ──────────────────────
// Each entry maps 1-to-1 with a REST endpoint.

const toolDefinitions: ToolDef[] = [
  // ── Inputs ──
  {
    method: "GET",
    path: "/api/v2/inputs",
    operationId: "listInputs",
    summary: "List all audio inputs",
    description:
      "Returns all inputs. By default only enabled inputs are returned; use includeDisabled=true to include disabled inputs.",
    parameters: [
      {
        name: "includeDisabled",
        in: "query",
        description: "Include disabled inputs in the response",
        required: false,
        schema: { type: "boolean" },
      },
    ],
    pathTemplate: "/api/v2/inputs",
  },
  {
    method: "GET",
    path: "/api/v2/inputs/{inputId}",
    operationId: "getInput",
    summary: "Get a single input by ID",
    description: "Returns the input if it exists.",
    parameters: [
      {
        name: "inputId",
        in: "path",
        description: "Unique input identifier",
        required: true,
        schema: { type: "string" },
      },
    ],
    pathTemplate: "/api/v2/inputs/{inputId}",
  },
  {
    method: "POST",
    path: "/api/v2/inputs",
    operationId: "createInput",
    summary: "Create a new ephemeral audio input",
    description:
      "Registers an ephemeral audio input that can be routed to outputs. Returns 201 with a Location header pointing to the created resource.",
    parameters: [],
    requestBody: {
      required: true,
      contentType: "application/json",
      schemaRef: "CreateInputRequest",
    },
    pathTemplate: "/api/v2/inputs",
  },
  {
    method: "DELETE",
    path: "/api/v2/inputs/{inputId}",
    operationId: "deleteInput",
    summary: "Delete an ephemeral audio input",
    description:
      "Removes a previously registered ephemeral input. Static (YAML-configured) inputs cannot be deleted.",
    parameters: [
      {
        name: "inputId",
        in: "path",
        description: "Unique input identifier",
        required: true,
        schema: { type: "string" },
      },
    ],
    pathTemplate: "/api/v2/inputs/{inputId}",
  },
  {
    method: "PUT",
    path: "/api/v2/inputs/{inputId}/enabled",
    operationId: "setInputEnabled",
    summary: "Set input enabled state",
    description:
      "Enables or disables an audio input. A disabled input remains registered but is unavailable for new route creation. Existing active routes are unaffected.",
    parameters: [
      {
        name: "inputId",
        in: "path",
        description: "Unique input identifier",
        required: true,
        schema: { type: "string" },
      },
    ],
    requestBody: {
      required: true,
      contentType: "application/json",
      schemaRef: "EnabledRequest",
    },
    pathTemplate: "/api/v2/inputs/{inputId}/enabled",
  },

  // ── Outputs ──
  {
    method: "GET",
    path: "/api/v2/outputs",
    operationId: "listOutputs",
    summary: "List all audio outputs",
    description:
      "Returns all outputs with current volume and mute state. By default only enabled outputs are returned.",
    parameters: [
      {
        name: "includeDisabled",
        in: "query",
        description: "Include disabled outputs in the response",
        required: false,
        schema: { type: "boolean" },
      },
    ],
    pathTemplate: "/api/v2/outputs",
  },
  {
    method: "GET",
    path: "/api/v2/outputs/{outputId}",
    operationId: "getOutput",
    summary: "Get a single output by ID",
    description: "Returns the output with current volume and mute state.",
    parameters: [
      {
        name: "outputId",
        in: "path",
        description: "Unique output identifier",
        required: true,
        schema: { type: "string" },
      },
    ],
    pathTemplate: "/api/v2/outputs/{outputId}",
  },
  {
    method: "PUT",
    path: "/api/v2/outputs/{outputId}/volume",
    operationId: "setOutputVolume",
    summary: "Set output volume",
    description:
      "Sets the volume level for an audio output. Returns the applied volume in the response.",
    parameters: [
      {
        name: "outputId",
        in: "path",
        description: "Unique output identifier",
        required: true,
        schema: { type: "string" },
      },
    ],
    requestBody: {
      required: true,
      contentType: "application/json",
      schemaRef: "VolumeRequest",
    },
    pathTemplate: "/api/v2/outputs/{outputId}/volume",
  },
  {
    method: "PUT",
    path: "/api/v2/outputs/{outputId}/mute",
    operationId: "setOutputMute",
    summary: "Set output mute state",
    description:
      "Mutes or unmutes an audio output. Returns the applied mute state.",
    parameters: [
      {
        name: "outputId",
        in: "path",
        description: "Unique output identifier",
        required: true,
        schema: { type: "string" },
      },
    ],
    requestBody: {
      required: true,
      contentType: "application/json",
      schemaRef: "MuteRequest",
    },
    pathTemplate: "/api/v2/outputs/{outputId}/mute",
  },
  {
    method: "PUT",
    path: "/api/v2/outputs/{outputId}/enabled",
    operationId: "setOutputEnabled",
    summary: "Set output enabled state",
    description:
      "Enables or disables an audio output. A disabled output remains registered but is unavailable for new route creation. Existing active routes are unaffected.",
    parameters: [
      {
        name: "outputId",
        in: "path",
        description: "Unique output identifier",
        required: true,
        schema: { type: "string" },
      },
    ],
    requestBody: {
      required: true,
      contentType: "application/json",
      schemaRef: "EnabledRequest",
    },
    pathTemplate: "/api/v2/outputs/{outputId}/enabled",
  },

  // ── Groups ──
  {
    method: "GET",
    path: "/api/v2/groups",
    operationId: "listGroups",
    summary: "List all audio output groups",
    description:
      "Returns all groups with their member output IDs. By default only enabled groups are returned.",
    parameters: [
      {
        name: "includeDisabled",
        in: "query",
        description: "Include disabled groups in the response",
        required: false,
        schema: { type: "boolean" },
      },
    ],
    pathTemplate: "/api/v2/groups",
  },
  {
    method: "GET",
    path: "/api/v2/groups/{groupId}",
    operationId: "getGroup",
    summary: "Get a single group by ID",
    description: "Returns the group with its member output IDs.",
    parameters: [
      {
        name: "groupId",
        in: "path",
        description: "Unique group identifier",
        required: true,
        schema: { type: "string" },
      },
    ],
    pathTemplate: "/api/v2/groups/{groupId}",
  },
  {
    method: "PUT",
    path: "/api/v2/groups/{groupId}/volume",
    operationId: "setGroupVolume",
    summary: "Set volume for all outputs in a group",
    description:
      "Sets the volume level for every output in the specified group.",
    parameters: [
      {
        name: "groupId",
        in: "path",
        description: "Unique group identifier",
        required: true,
        schema: { type: "string" },
      },
    ],
    requestBody: {
      required: true,
      contentType: "application/json",
      schemaRef: "VolumeRequest",
    },
    pathTemplate: "/api/v2/groups/{groupId}/volume",
  },
  {
    method: "PUT",
    path: "/api/v2/groups/{groupId}/mute",
    operationId: "setGroupMute",
    summary: "Mute or unmute all outputs in a group",
    description:
      "Sets the mute state for every output in the specified group. Idempotent — setting the same state returns 200 OK with the current state.",
    parameters: [
      {
        name: "groupId",
        in: "path",
        description: "Unique group identifier",
        required: true,
        schema: { type: "string" },
      },
    ],
    requestBody: {
      required: true,
      contentType: "application/json",
      schemaRef: "MuteRequest",
    },
    pathTemplate: "/api/v2/groups/{groupId}/mute",
  },
  {
    method: "PUT",
    path: "/api/v2/groups/{groupId}/enabled",
    operationId: "setGroupEnabled",
    summary: "Set group enabled state",
    description:
      "Enables or disables an output group. A disabled group remains registered but is unavailable for new route creation. Existing active routes are unaffected.",
    parameters: [
      {
        name: "groupId",
        in: "path",
        description: "Unique group identifier",
        required: true,
        schema: { type: "string" },
      },
    ],
    requestBody: {
      required: true,
      contentType: "application/json",
      schemaRef: "EnabledRequest",
    },
    pathTemplate: "/api/v2/groups/{groupId}/enabled",
  },

  // ── Routes ──
  {
    method: "GET",
    path: "/api/v2/routes",
    operationId: "listRoutes",
    summary: "List all active audio routes",
    description:
      "Returns all current routes with their input, target, and status. Supports optional query parameters for filtering by status, inputId, and targetId (AND logic).",
    parameters: [
      {
        name: "status",
        in: "query",
        description: "Filter by route status",
        required: false,
        schema: {
          type: "string",
          enum: ["STARTING", "ACTIVE", "STOPPING", "STOPPED", "FAILED"],
        },
      },
      {
        name: "inputId",
        in: "query",
        description: "Filter by source input ID",
        required: false,
        schema: { type: "string" },
      },
      {
        name: "targetId",
        in: "query",
        description: "Filter by target output or group ID",
        required: false,
        schema: { type: "string" },
      },
    ],
    pathTemplate: "/api/v2/routes",
  },
  {
    method: "GET",
    path: "/api/v2/routes/{routeId}",
    operationId: "getRoute",
    summary: "Get a single route by ID",
    description: "Returns the route with its current status and timestamps.",
    parameters: [
      {
        name: "routeId",
        in: "path",
        description: "Unique route identifier",
        required: true,
        schema: { type: "string" },
      },
    ],
    pathTemplate: "/api/v2/routes/{routeId}",
  },
  {
    method: "POST",
    path: "/api/v2/routes",
    operationId: "createRoute",
    summary: "Create a new audio route",
    description:
      "Routes audio from an input to a target output or group. Resolves the target based on targetType (SINGLE_OUTPUT or OUTPUT_GROUP).",
    parameters: [],
    requestBody: {
      required: true,
      contentType: "application/json",
      schemaRef: "CreateRouteRequest",
    },
    pathTemplate: "/api/v2/routes",
  },
  {
    method: "DELETE",
    path: "/api/v2/routes/{routeId}",
    operationId: "deleteRoute",
    summary: "Stop and delete a route",
    description:
      "Stops audio playback and removes the route. Ephemeral inputs with autoRemove=true are cleaned up automatically.",
    parameters: [
      {
        name: "routeId",
        in: "path",
        description: "Unique route identifier",
        required: true,
        schema: { type: "string" },
      },
    ],
    pathTemplate: "/api/v2/routes/{routeId}",
  },
  {
    method: "POST",
    path: "/api/v2/routes/{routeId}/transfer",
    operationId: "transferRoute",
    summary: "Transfer an active route to a new output or group",
    description:
      "Seamlessly moves audio playback to a new target without interruption. The old route is replaced — callers must update their references to the new routeId.",
    parameters: [
      {
        name: "routeId",
        in: "path",
        description: "Unique route identifier",
        required: true,
        schema: { type: "string" },
      },
    ],
    requestBody: {
      required: true,
      contentType: "application/json",
      schemaRef: "TransferRequest",
    },
    pathTemplate: "/api/v2/routes/{routeId}/transfer",
  },
  {
    method: "PUT",
    path: "/api/v2/routes/{routeId}/pause",
    operationId: "setRoutePause",
    summary: "Set pause state on a route",
    description:
      "Pauses or resumes audio playback for an active route. Set paused=true to pause, paused=false to resume. Idempotent — setting the same state returns 200 OK with the current state.",
    parameters: [
      {
        name: "routeId",
        in: "path",
        description: "Unique route identifier",
        required: true,
        schema: { type: "string" },
      },
    ],
    requestBody: {
      required: true,
      contentType: "application/json",
      schemaRef: "PauseRequest",
    },
    pathTemplate: "/api/v2/routes/{routeId}/pause",
  },

  // ── Playback ──
  {
    method: "POST",
    path: "/api/v2/play",
    operationId: "playback",
    summary: "Play audio from a URI",
    description:
      "Provides instant playback: supply a URI and target, and the system automatically creates an ephemeral input and route. Supports YouTube, SoundCloud, and other URI schemes. If volume is provided, it is set before playback starts.",
    parameters: [],
    requestBody: {
      required: true,
      contentType: "application/json",
      schemaRef: "PlaybackRequest",
    },
    pathTemplate: "/api/v2/play",
  },

  // ── Master Mute ──
  {
    method: "GET",
    path: "/api/v2/master-mute",
    operationId: "getMasterMute",
    summary: "Get master mute state",
    description:
      "Returns the current global mute state. When active, all audio outputs are silenced.",
    parameters: [],
    pathTemplate: "/api/v2/master-mute",
  },
  {
    method: "PUT",
    path: "/api/v2/master-mute",
    operationId: "setMasterMute",
    summary: "Set master mute state",
    description:
      "Sets the global mute state. PUT is idempotent — same request repeated produces the same result.",
    parameters: [],
    requestBody: {
      required: true,
      contentType: "application/json",
      schemaRef: "MuteRequest",
    },
    pathTemplate: "/api/v2/master-mute",
  },
];

// ─── Zod schema builder ────────────────────────────────────────────────────

function buildZodSchema(tool: ToolDef): ZodRawShape {
  const fields: Record<string, z.ZodTypeAny> = {};

  // Path & query parameters
  for (const param of tool.parameters) {
    let field: z.ZodTypeAny;
    if (param.schema.enum) {
      field = z.enum(param.schema.enum as [string, ...string[]]).describe(param.description);
    } else {
      switch (param.schema.type) {
        case "boolean":
          field = z.boolean().describe(param.description);
          break;
        case "integer":
        case "number":
          field = z.number().describe(param.description);
          break;
        default:
          field = z.string().describe(param.description);
      }
    }
    if (!param.required) {
      field = field.optional();
    }
    fields[param.name] = field;
  }

  // Request body fields
  if (tool.requestBody) {
    const bodyFields = requestBodySchemas[tool.requestBody.schemaRef];
    if (bodyFields) {
      for (const [name, field] of Object.entries(bodyFields)) {
        fields[name] = field;
      }
    }
  }

  return fields;
}

// ─── Request body schema definitions ────────────────────────────────────────

const requestBodySchemas: Record<string, Record<string, z.ZodTypeAny>> = {
  CreateInputRequest: {
    inputId: z
      .string()
      .min(1)
      .regex(/^[a-zA-Z0-9\-_]{1,255}$/)
      .describe("Unique identifier for the input (alphanumeric, hyphens, underscores, 1-255 chars)"),
    displayName: z.string().min(1).describe("Human-readable display name"),
    uri: z.string().min(1).describe("Source URI (http://, file://, alsa://, signal://, etc.)"),
    enabled: z.boolean().optional().describe("Whether input is active (defaults to true)"),
    autoRemove: z.boolean().optional().describe("Auto-remove input when its route is stopped (defaults to false)"),
  },
  EnabledRequest: {
    enabled: z.boolean().describe("Whether the input/output/group should be enabled"),
  },
  VolumeRequest: {
    volume: z.number().int().min(0).max(100).describe("Volume level 0-100 (inclusive)"),
  },
  MuteRequest: {
    muted: z.boolean().describe("true to mute, false to unmute"),
  },
  PauseRequest: {
    paused: z.boolean().describe("true to pause playback, false to resume playback"),
  },
  CreateRouteRequest: {
    inputId: z.string().min(1).describe("Source audio input ID"),
    targetId: z.string().min(1).describe("ID of the destination output or group"),
    targetType: z
      .enum(["SINGLE_OUTPUT", "OUTPUT_GROUP"])
      .describe("Target type: SINGLE_OUTPUT or OUTPUT_GROUP"),
  },
  TransferRequest: {
    targetId: z.string().min(1).describe("ID of the destination output or group"),
    targetType: z
      .enum(["SINGLE_OUTPUT", "OUTPUT_GROUP"])
      .describe("Target type: SINGLE_OUTPUT or OUTPUT_GROUP"),
  },
  PlaybackRequest: {
    uri: z.string().min(1).describe("Audio source URI (http://, file://, etc.)"),
    targetId: z.string().min(1).describe("ID of the destination output or group"),
    targetType: z
      .enum(["SINGLE_OUTPUT", "OUTPUT_GROUP"])
      .describe("Target type: SINGLE_OUTPUT or OUTPUT_GROUP"),
    displayName: z
      .string()
      .optional()
      .describe("Optional friendly name for the ephemeral input"),
    volume: z
      .number()
      .int()
      .min(0)
      .max(100)
      .optional()
      .describe("Optional volume level (0-100), set before playback starts"),
  },
};

// ─── HTTP helper ────────────────────────────────────────────────────────────

async function callMultiroomApi(
  baseUrl: string,
  method: string,
  path: string,
  body?: unknown,
): Promise<string> {
  const url = `${baseUrl}${path}`;

  const init: RequestInit = {
    method,
    headers: { "Content-Type": "application/json", Accept: "application/json" },
  };

  if (body !== undefined) {
    init.body = JSON.stringify(body);
  }

  const response = await fetch(url, init);

  // 204 No Content (e.g. DELETE responses)
  if (response.status === 204) {
    return JSON.stringify({ success: true, status: 204 });
  }

  const text = await response.text();

  if (!response.ok) {
    // Pass through RFC 7807 error details
    let detail = text;
    try {
      const err = JSON.parse(text);
      detail = err.detail || err.title || text;
    } catch {
      // use raw text
    }
    throw new Error(`Multiroom API error ${response.status}: ${detail}`);
  }

  return text || JSON.stringify({ success: true, status: response.status });
}

// ─── Path builder ───────────────────────────────────────────────────────────

function resolvePath(template: string, args: Record<string, unknown>): string {
  let path = template;
  for (const [key, value] of Object.entries(args)) {
    if (typeof value === "string" || typeof value === "number") {
      path = path.replace(`{${key}}`, encodeURIComponent(String(value)));
    }
  }
  return path;
}

// ─── Tool registration ──────────────────────────────────────────────────────

export function registerTools(server: McpServer, multiroomUrl: string): void {
  for (const tool of toolDefinitions) {
    const schema = buildZodSchema(tool);

    server.registerTool(
      tool.operationId,
      {
        title: tool.summary,
        description: tool.summary,
        inputSchema: schema as any, // eslint-disable-line @typescript-eslint/no-explicit-any
      },
      async (args: Record<string, unknown>): Promise<CallToolResult> => {
        try {
          // Separate path params, query params, and body fields
          const pathParams: Record<string, string> = {};
          const queryParams: Record<string, string> = {};

          for (const param of tool.parameters) {
            const value = args[param.name];
            if (value === undefined || value === null) continue;
            if (param.in === "path") {
              pathParams[param.name] = String(value);
            } else if (param.in === "query") {
              queryParams[param.name] = String(value);
            }
          }

          let path = resolvePath(tool.pathTemplate, pathParams);

          // Append query parameters
          const queryEntries = Object.entries(queryParams).filter(
            ([, v]) => v !== undefined && v !== "",
          );
          if (queryEntries.length > 0) {
            const qs = new URLSearchParams(queryEntries).toString();
            path = `${path}?${qs}`;
          }

          // Build request body
          let body: unknown = undefined;
          if (tool.requestBody) {
            const bodySchema = requestBodySchemas[tool.requestBody.schemaRef];
            if (bodySchema) {
              body = {};
              for (const key of Object.keys(bodySchema)) {
                const value = args[key];
                if (value !== undefined && value !== null) {
                  (body as Record<string, unknown>)[key] = value;
                }
              }
              if (Object.keys(body as Record<string, unknown>).length === 0) {
                body = undefined;
              }
            }
          }

          const result = await callMultiroomApi(
            multiroomUrl,
            tool.method,
            path,
            body,
          );

          return {
            content: [{ type: "text", text: result }],
          };
        } catch (error) {
          const message =
            error instanceof Error ? error.message : String(error);
          return {
            content: [{ type: "text", text: `Error: ${message}` }],
            isError: true,
          };
        }
      },
    );
  }

  console.log(`Registered ${toolDefinitions.length} MCP tools`);
}
