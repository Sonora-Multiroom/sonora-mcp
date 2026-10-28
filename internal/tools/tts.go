package tools

import (
	"context"
	"errors"
	"net/http"

	"github.com/Sonora-Multiroom/sonora-cli/hub"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ttsError replaces a "not offered" answer from the hub's TTS endpoints with
// the shared client's diagnosis of why text-to-speech is unavailable; it costs
// one extension-inventory call and only follows a failure. Any other error is
// returned unchanged.
func ttsError(ctx context.Context, client *http.Client, hubURL string, err error) error {
	var notOffered *hub.TTSNotOfferedError
	if errors.As(err, &notOffered) {
		return hub.DiagnoseTTSUnavailable(ctx, client, hubURL)
	}
	return err
}

type speakIn struct {
	Text           string   `json:"text" jsonschema:"The text to speak"`
	TargetName     string   `json:"targetName" jsonschema:"ID of the output or group to speak on, the same IDs as targetId in the other tools, not a display name"`
	TargetType     string   `json:"targetType" jsonschema:"SINGLE_OUTPUT if targetName is an output, OUTPUT_GROUP if it is a group"`
	ProviderName   *string  `json:"providerName,omitempty" jsonschema:"Text-to-speech provider, e.g. piper-local (runs on the hub, free); when omitted the hub's configured default provider is used, which may be a billed cloud service"`
	Voice          *string  `json:"voice,omitempty" jsonschema:"Voice short name, as listed by listTtsVoices for the provider; the hub's configured voice when omitted"`
	Language       *string  `json:"language,omitempty" jsonschema:"Language code, e.g. uk-UA; the hub's configured language when omitted"`
	PlaybackMode   *string  `json:"playbackMode,omitempty" jsonschema:"How the announcement plays over other audio: duck-others (other audio keeps playing quieter underneath) or mix (it plays on unchanged); the hub rejects any other value; the extension's configured mode when omitted"`
	RouteVolume    *int     `json:"routeVolume,omitempty" jsonschema:"Volume of this announcement only, an integer from 0 to 100; it changes no output's volume; the room's volume applies when omitted"`
	DuckingLevelDb *float64 `json:"duckingLevelDb,omitempty" jsonschema:"With duck-others, how far other audio is lowered, from -60 to 0 dB; the hub rejects it together with mix; the configured level when omitted"`
}

type listVoicesIn struct {
	ProviderName string  `json:"providerName" jsonschema:"Text-to-speech provider whose voices to list, e.g. piper-local"`
	Language     *string `json:"language,omitempty" jsonschema:"Only voices for this language code, e.g. uk-UA; all languages when omitted"`
	Engine       *string `json:"engine,omitempty" jsonschema:"Only voices of this engine; all engines when omitted"`
}

type clearCacheIn struct {
	ProviderName *string `json:"providerName,omitempty" jsonschema:"Clear only this provider's cached audio; the whole cache when omitted"`
}

// cacheCleared confirms a cleared cache; the hub answers 204 with no body.
type cacheCleared struct {
	Cleared      bool    `json:"cleared" jsonschema:"Always true"`
	ProviderName *string `json:"providerName,omitempty" jsonschema:"The provider whose cache was cleared; absent when the whole cache was cleared"`
}

// withTimeout bounds one TTS call, and the diagnosis that may follow it, by
// the timeout of the client that makes the call.
func withTimeout(ctx context.Context, c *http.Client) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, c.Timeout)
}

// registerTTS registers the text-to-speech tools. speak uses speakClient,
// whose timeout allows a slow provider to accept an announcement.
func registerTTS(s *mcp.Server, client, speakClient *http.Client, hubURL string) int {
	add(s, toolSpec{
		Name: "speak",
		Description: "Speak a text aloud on an output or group through the hub's text-to-speech. Changes state: each call queues another announcement. " +
			"Returns once the announcement is queued, not once it has been spoken. " +
			"targetName takes the same IDs as targetId in the other tools (see listOutputs and listGroups), not display names. " +
			"Omitting providerName uses the hub's configured default provider, which may be a billed cloud service; piper-local runs on the hub at no cost. " +
			"Use listTtsVoices to find a voice to pass as voice. " +
			"playbackMode is duck-others (other audio keeps playing quieter underneath) or mix (the announcement plays over it unchanged); duckingLevelDb applies only with duck-others. " +
			"routeVolume sets this announcement's volume only and changes no output's volume. " +
			"Returns {\"announcementId\", \"cacheHit\", \"queueDepth\"}: the announcement's ID, whether its audio came from the hub's cache, and how many announcements are queued; " +
			"fails with NotFound if the target or provider is unknown, Validation if the hub rejects the request, ServiceUnavailable if the provider is down, slow or rate-limited, Timeout if the hub did not answer in time (the announcement may still have been queued and spoken, so check before repeating the call; a phrase the hub has already synthesised is accepted at once), and TtsUnavailable if the hub has no working text-to-speech (the message says why).",
		Kind: StateChanging, Method: "POST", Path: "/api/tts/speak",
	}, inputSchema[speakIn](withMinLength("text", 1), withMinLength("targetName", 1), withEnum("targetType", targetTypes...),
		withRange("routeVolume", 0, 100), withRange("duckingLevelDb", -60, 0)),
		func(ctx context.Context, in speakIn) (*hub.SpeakAccepted, error) {
			ctx, cancel := withTimeout(ctx, speakClient)
			defer cancel()
			res, err := hub.Speak(ctx, speakClient, hubURL, hub.SpeakRequest{
				Text: in.Text, TargetName: in.TargetName, TargetType: in.TargetType,
				ProviderName: in.ProviderName, Voice: in.Voice, Language: in.Language,
				PlaybackMode: in.PlaybackMode, RouteVolume: in.RouteVolume, DuckingLevelDb: in.DuckingLevelDb,
			})
			if err != nil {
				return nil, ttsError(ctx, client, hubURL, err)
			}
			return res, nil
		})

	add(s, toolSpec{
		Name: "listTtsVoices",
		Description: "List the voices a text-to-speech provider offers, optionally only for one language or engine. Read-only. " +
			"Pass a voice's shortName to speak as voice. Some providers, such as piper-local, cannot list voices: the hub then answers Validation (\"does not support voice listing\"). " +
			"Returns {\"providerName\", \"voices\": [...]} with each voice's shortName, fullName, engine, language and gender (null where the provider does not say); an empty list is not an error; " +
			"fails with NotFound if the provider is unknown and TtsUnavailable if the hub has no working text-to-speech (the message says why).",
		Kind: ReadOnly, Method: "GET", Path: "/api/tts/providers/{providerName}/voices",
	}, inputSchema[listVoicesIn](withMinLength("providerName", 1)),
		func(ctx context.Context, in listVoicesIn) (*hub.TTSVoiceList, error) {
			ctx, cancel := withTimeout(ctx, client)
			defer cancel()
			res, err := hub.ListTTSVoices(ctx, client, hubURL, in.ProviderName, in.Language, in.Engine)
			if err != nil {
				return nil, ttsError(ctx, client, hubURL, err)
			}
			return res, nil
		})

	add(s, toolSpec{
		Name: "getTtsCacheStats",
		Description: "Get statistics of the hub's text-to-speech audio cache. Read-only. " +
			"Returns {\"totalEntries\", \"totalSizeBytes\", \"maxSizeBytes\", \"entriesByProvider\"}: sizes are in bytes and entriesByProvider maps each provider name to its entries; " +
			"fails with TtsUnavailable if the hub has no working text-to-speech (the message says why).",
		Kind: ReadOnly, Method: "GET", Path: "/api/tts/cache/stats",
	}, inputSchema[noInput](),
		func(ctx context.Context, _ noInput) (*hub.TTSCacheStats, error) {
			ctx, cancel := withTimeout(ctx, client)
			defer cancel()
			res, err := hub.GetTTSCacheStats(ctx, client, hubURL)
			if err != nil {
				return nil, ttsError(ctx, client, hubURL, err)
			}
			return res, nil
		})

	add(s, toolSpec{
		Name: "clearTtsCache",
		Description: "Clear the hub's text-to-speech audio cache: every provider's cached audio, or only providerName's. Destructive but idempotent: clearing an empty cache changes nothing. " +
			"Use it after the configured voice changes, when old phrases still play in the old voice. The next announcement of a cleared phrase is synthesised again, and a billed provider charges again. " +
			"Returns {\"cleared\": true} plus providerName when one was given; " +
			"fails with NotFound if the provider is unknown and TtsUnavailable if the hub has no working text-to-speech (the message says why).",
		Kind: DestructiveIdempotent, Method: "DELETE", Path: "/api/tts/cache",
	}, inputSchema[clearCacheIn](),
		func(ctx context.Context, in clearCacheIn) (*cacheCleared, error) {
			ctx, cancel := withTimeout(ctx, client)
			defer cancel()
			if err := hub.ClearTTSCache(ctx, client, hubURL, in.ProviderName); err != nil {
				return nil, ttsError(ctx, client, hubURL, err)
			}
			return &cacheCleared{Cleared: true, ProviderName: in.ProviderName}, nil
		})

	return 4
}
