# gojobs

Go CLI that reads a job page URL, grounds Gemma 4 with a structured candidate dossier, and returns a tailored introduction message you can send in an application or cold outreach.

## What it does

- Fetches and cleans the job page text from the URL you pass.
- Loads a structured profile for Carlos Eduardo Gonzalez Henriquez built from:
  - curated CV facts captured in `profiles/carlos_gonzalez.json`
  - public GitHub profile `cegh2001`
  - public repos such as `godojo`, `siberiano`, and `openai-devassistant`
  - verified private-repository signals gathered through authenticated GitHub access
- Calls the official Google Gen AI Go SDK with Gemma 4.
- Forces structured JSON output so the result is stable and easy to evolve.

## Models

This bootstrap follows the current Google AI SDK and model guidance checked during implementation:

- `gemma-4-31b-it` as the default model for normal runs
- `-mode fast` is kept as a compatibility alias, but it still defaults to `gemma-4-31b-it`
- the main latency optimization comes from the compact prompt, not from switching to a smaller model
- SDK: `google.golang.org/genai`

## Quick start

Set your API key in `.env.local`, `.env`, or the shell environment using either `GEMINI_API_KEY` or `GOOGLE_API_KEY`.

```bash
go run ./cmd/gojobs -url https://www.workatastartup.com/jobs/89001
```

Use the optimized default path explicitly:

```bash
go run ./cmd/gojobs -url https://www.workatastartup.com/jobs/89001 -mode fast
```

Inject extra context that is true for a specific application:

```bash
go run ./cmd/gojobs \
  -url https://www.workatastartup.com/jobs/89001 \
  -note "Open to relocating for the right founding role."
```

Inspect the exact prompt without calling Google AI:

```bash
go run ./cmd/gojobs -url https://www.workatastartup.com/jobs/89001 -prompt-only
```

Print raw JSON instead of the human-readable report:

```bash
go run ./cmd/gojobs -url https://www.workatastartup.com/jobs/89001 -json
```

## Flags

- `-url`: job page URL to analyze
- `-profile`: candidate profile JSON path, default `profiles/carlos_gonzalez.json`
- `-mode`: `heavy` or `fast` for compatibility; both default to the same optimized `gemma-4-31b-it` path unless `-model` is provided
- `-model`: explicit model override
- `-note`: extra candidate context you want the model to consider
- `-prompt-only`: print the constructed prompt and exit
- `-json`: print raw structured JSON
- `-timeout`: total timeout for page fetch and model call

## Current limits

- The profile is curated from verified sources already present in this repo plus public GitHub signals. It is not yet auto-refreshed from PDFs or the GitHub API at runtime.
- Public GitHub did not expose detailed public repository contributions for Gonavi or LegalContigo, so company-specific claims rely on the CV material unless you pass extra runtime notes.
- If a page is heavily JS-rendered and the server response is too thin, the fetcher may need a browser-backed fallback in a later iteration.

## Validation

Current bootstrap validation:

```bash
go test ./...
```
