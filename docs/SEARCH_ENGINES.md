# Search Engine Integration

Nebula now supports multiple search engines for OSINT gathering, with both free and paid options.

## Free Search Engines (No API Key Required)

### DuckDuckGo (Primary)
- **Status**: Enabled by default
- **Pros**: Privacy-focused, no CAPTCHA issues, reliable
- **Endpoint**: `https://lite.duckduckgo.com/lite/`
- **Usage**: Automatically used as the primary search engine

### Bing (Secondary)
- **Status**: Enabled by default  
- **Pros**: Good coverage, often returns different results than DuckDuckGo
- **Endpoint**: `https://www.bing.com/search`
- **Usage**: Automatically used as a secondary search engine to complement DuckDuckGo

## Paid Search Engine (API Key Required)

### Google Custom Search API
- **Status**: Optional (requires API key)
- **Pros**: Highest quality results, most comprehensive index
- **Cons**: Paid service (100 free queries/day, then $5 per 1000 queries)
- **Setup**:
  1. Get API key from [Google Cloud Console](https://console.cloud.google.com/)
  2. Create a Custom Search Engine at [cse.google.com](https://cse.google.com/cse/)
  3. Add to your `.env` file:
     ```
     GOOGLE_CUSTOM_SEARCH_API_KEY=your_api_key
     GOOGLE_CUSTOM_SEARCH_ENGINE_ID=your_engine_id
     ```

## What Changed

### Removed
- **Direct Google Scraping**: Too many CAPTCHAs, unreliable
- **Mojeek**: Returned too much noise/irrelevant results

### Added
- **Bing Search**: New free search engine with good coverage
- **Smart Fallback**: System uses DuckDuckGo + Bing by default, Google API only if credentials provided

## How It Works

The search engine collector now:
1. Builds targeted queries based on the input type (email, username, person name, etc.)
2. Searches DuckDuckGo first (free, reliable)
3. Searches Bing second (free, additional coverage)
4. Searches Google API last (only if API credentials are provided)
5. Deduplicates results across all engines
6. Extracts additional intelligence (names, social profiles) from search results

## Configuration

No configuration needed for free search engines - they work out of the box!

For Google API (optional):
```bash
cp .env.example .env
# Edit .env and add your Google API credentials
```

## Example Usage

```bash
# Search for a person (uses free engines by default)
./nebula search --type person_name --query "John Doe"

# Search with Google API (if configured)
# Results will include DuckDuckGo + Bing + Google API
./nebula search --type email --query "john@example.com"
```
