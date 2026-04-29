package persistence

import "golinks/internal/core/links"

// toDomainShortcut converts a DB row to the domain entity.
func toDomainShortcut(r shortcutRow) *links.Shortcut {
	return &links.Shortcut{
		ID:        int(r.ID),
		Word:      r.Word,
		Link:      r.Link,
		User:      r.User,
		CreatedAt: r.CreatedAt,
	}
}

// fromDomainShortcut converts a domain entity to a DB row, ready for insert/update.
func fromDomainShortcut(s *links.Shortcut) shortcutRow {
	return shortcutRow{
		ID:        uint(s.ID),
		Word:      s.Word,
		Link:      s.Link,
		User:      s.User,
		CreatedAt: s.CreatedAt,
	}
}
