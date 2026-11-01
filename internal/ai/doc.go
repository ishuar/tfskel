// Package ai produces a human-readable narrative analysis of a Terraform plan
// by sending a sanitized, compressed view of the plan to an LLM provider.
//
// The package is layered so additional providers (or an MCP server surface) can
// be added later without touching command code:
//
//   - Client   is the provider-agnostic interface (Explain streams Markdown).
//   - Payload  is the wire-format shipped to the model (built from a parsed plan).
//   - Config   holds model + max-tokens settings (loaded from viper).
//
// Two providers ship: Anthropic (the default) and Gemini. NewClient picks one
// from the TFSKEL_AI_PROVIDER environment variable and rejects any other
// value. ANTHROPIC_API_KEY and GEMINI_API_KEY are the sole credential
// sources; keys are never read from config files or command-line flags.
package ai
