# Security

## Supported versions

Security fixes are applied to the latest minor release on the default branch (`main`). Release binaries are published via GitHub Releases and the npm package `@fatecannotbealtered-/cnstock-cli`.

## Reporting a vulnerability

Please **do not** file a public GitHub issue for undisclosed security vulnerabilities.

Instead, report privately via [GitHub Security Advisories](https://github.com/fatecannotbealtered/cnstock-cli/security/advisories/new) for this repository, or contact the maintainers through the contact options on the repository homepage.

Include:

- Description of the issue and impact
- Steps to reproduce (if safe to share)
- Affected versions or install methods (binary / npm / `go install`)

## Affiliation

**cnstock-cli is NOT an official Tencent product. It is not affiliated with, endorsed by, or sponsored by Tencent Holdings Limited or any of its subsidiaries.** The "Tencent Finance" name is used solely to describe the data source. All trademarks belong to their respective owners.

## Data rights

The MIT license applies only to the source code of this tool. **Market data retrieved through the endpoints remains the property of its respective rights holders.** This tool does not grant any rights to third-party data. Users are solely responsible for how they use the data and for complying with applicable laws, regulations, and terms of service.

## Data source disclaimer

**This tool is NOT based on any official Tencent Finance API.**

The endpoints used by cnstock-cli are web endpoints observed from Tencent Finance public web pages and clients. They are:

- **Not documented** —Tencent has not published any developer documentation for these endpoints
- **Not contracted** —There is no formal SLA, schema contract, or rate-limit policy
- **Not guaranteed** —Endpoints may change, return different data, or become unavailable at any time without notice
- **Not for production** —These endpoints are intended for browser-based human consumption, not programmatic access

### What this means for you

- **Schema may drift**: Field positions, counts, and formats may change without warning. The CLI includes field-count validation and emits `warnings` when the response looks shorter than expected, but this is best-effort.
- **Rate limits are unknown**: Tencent may throttle or block clients that make frequent programmatic requests. The CLI does not implement rate limiting —use responsibly.
- **Data may be incomplete**: Some markets or symbols may return partial data or errors depending on Tencent's backend state.
- **No uptime guarantee**: Endpoints may be temporarily or permanently unavailable.

### Acceptable use

- Personal stock lookup and research
- Learning, demos, and agent-assisted analysis
- Non-commercial, non-critical workflows

### NOT acceptable use

- Automated trading systems
- Commercial products or services
- Compliance-sensitive financial reporting
- High-frequency polling or scraping
- Any use where data accuracy or availability is critical

### Recommended alternatives for production use

For commercial products, trading systems, or compliance-sensitive workloads, use a licensed market data provider such as:

- Wind (万得)
- Tushare
- AKShare
- Bloomberg / Reuters
- Any provider with a published API, SLA, and data license

## Credential handling

This CLI does not require authentication. No API keys, tokens, or credentials are stored or transmitted. All requests are made to public web endpoints over HTTPS.
