package cnstockcli

import _ "embed"

// ChangelogMarkdown is embedded from the repository root so runtime changelog
// output, GitHub release notes, and human docs all share CHANGELOG.md as the
// single source of truth.
//
//go:embed CHANGELOG.md
var ChangelogMarkdown string
