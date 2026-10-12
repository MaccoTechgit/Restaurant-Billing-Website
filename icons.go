package main

import (
	"html/template"
	"strings"
)

// Line-icon set (24x24, stroke). Content refers to icons by name, or by the old emoji key which is mapped below.
var iconPaths = map[string]string{
	"receipt":   `<path d="M6 3h12v18l-3-2-3 2-3-2-3 2z"/><path d="M9 8h6M9 12h6"/>`,
	"bolt":      `<path d="M13 3 5 14h6l-1 7 8-11h-6z"/>`,
	"calc":      `<rect x="5" y="3" width="14" height="18" rx="2"/><path d="M8 7h8M8 12h2M12 12h2M8 16h2M12 16h2"/>`,
	"card":      `<rect x="3" y="6" width="18" height="12" rx="2"/><path d="M3 10h18M7 15h3"/>`,
	"printer":   `<path d="M7 9V3h10v6"/><rect x="4" y="9" width="16" height="8" rx="2"/><path d="M7 14h10v7H7z"/>`,
	"user":      `<circle cx="12" cy="8" r="4"/><path d="M4 21c0-4 4-6 8-6s8 2 8 6"/>`,
	"keyboard":  `<rect x="2" y="6" width="20" height="12" rx="2"/><path d="M6 10h.01M10 10h.01M14 10h.01M18 10h.01M7 14h10"/>`,
	"flame":     `<path d="M12 3c1 4 5 5 5 10a5 5 0 0 1-10 0c0-2 1-3 2-4 0 2 1 3 2 3 0-3-1-6 1-9z"/>`,
	"screen":    `<rect x="3" y="4" width="18" height="12" rx="2"/><path d="M8 20h8M12 16v4"/>`,
	"bell":      `<path d="M6 16v-5a6 6 0 0 1 12 0v5l2 2H4z"/><path d="M10 21h4"/>`,
	"sparkle":   `<path d="m12 3 2 5 5 2-5 2-2 5-2-5-5-2 5-2z"/>`,
	"volume":    `<path d="M4 10v4h4l5 4V6L8 10z"/><path d="M16 9a4 4 0 0 1 0 6"/>`,
	"phone":     `<rect x="7" y="2" width="10" height="20" rx="2"/><path d="M11 18h2"/>`,
	"table":     `<rect x="3" y="6" width="18" height="4" rx="1"/><path d="M6 10v9M18 10v9"/>`,
	"shield":    `<path d="M12 3 4 6v6c0 5 3.5 8 8 9 4.5-1 8-4 8-9V6z"/><path d="m9 12 2 2 4-4"/>`,
	"eye":       `<path d="M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12z"/><circle cx="12" cy="12" r="3"/>`,
	"box":       `<path d="m3 7 9-4 9 4v10l-9 4-9-4z"/><path d="m3 7 9 4 9-4M12 11v10"/>`,
	"trend":     `<path d="m3 7 6 6 4-4 8 8"/><path d="M15 17h6v-6"/>`,
	"alert":     `<path d="m12 3 10 18H2z"/><path d="M12 10v5M12 18h.01"/>`,
	"truck":     `<rect x="2" y="7" width="12" height="9" rx="1"/><path d="M14 10h4l3 3v3h-7"/><circle cx="7" cy="18" r="2"/><circle cx="17" cy="18" r="2"/>`,
	"trash":     `<path d="M4 7h16M9 7V4h6v3M6 7l1 14h10l1-14M10 11v6M14 11v6"/>`,
	"store":     `<path d="m3 9 2-5h14l2 5M3 9v11h18V9M3 9h18M9 20v-6h6v6"/>`,
	"clipboard": `<rect x="6" y="4" width="12" height="17" rx="2"/><rect x="9" y="2" width="6" height="4" rx="1"/><path d="M9 11h6M9 15h6"/>`,
	"book":      `<path d="M5 4h12a2 2 0 0 1 2 2v14H7a2 2 0 0 1-2-2z"/><path d="M5 18a2 2 0 0 1 2-2h12M9 8h6"/>`,
	"cog":       `<circle cx="12" cy="12" r="3"/><path d="M12 2v3M12 19v3M2 12h3M19 12h3M5 5l2 2M17 17l2 2M19 5l-2 2M7 17l-2 2"/>`,
	"chart":     `<path d="M4 20V4M4 20h16"/><path d="M8 16v-4M12 16V8M16 16v-6"/>`,
	"wallet":    `<rect x="3" y="6" width="18" height="14" rx="2"/><path d="M3 10h18"/><circle cx="16" cy="15" r="1"/>`,
	"lock":      `<rect x="5" y="11" width="14" height="10" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/>`,
	"upload":    `<path d="M12 20V9M7 14l5-5 5 5M5 4h14"/>`,
	"download":  `<path d="M12 4v11M7 10l5 5 5-5M5 20h14"/>`,
	"clock":     `<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>`,
	"users":     `<circle cx="9" cy="8" r="3"/><circle cx="17" cy="9" r="2.5"/><path d="M3 20c0-3 3-5 6-5s6 2 6 5M16 15c3 0 5 2 5 5"/>`,
	"calendar":  `<rect x="4" y="5" width="16" height="16" rx="2"/><path d="M4 10h16M8 3v4M16 3v4"/>`,
	"briefcase": `<rect x="3" y="7" width="18" height="13" rx="2"/><path d="M9 7V4h6v3M3 13h18"/>`,
	"key":       `<circle cx="8" cy="15" r="4"/><path d="m11 12 9-9M16 7l3 3"/>`,
	"pin":       `<path d="M12 21s7-6 7-12a7 7 0 0 0-14 0c0 6 7 12 7 12z"/><circle cx="12" cy="9" r="2.5"/>`,
	"doc":       `<path d="M7 3h8l4 4v14H7z"/><path d="M15 3v4h4M10 12h6M10 16h6"/>`,
	"bike":      `<circle cx="6" cy="17" r="3"/><circle cx="18" cy="17" r="3"/><path d="M6 17h6l3-7h3M13 10H9l-2 7"/>`,
	"call":      `<path d="M5 4h4l2 5-2 1a11 11 0 0 0 5 5l1-2 5 2v4a2 2 0 0 1-2 2A16 16 0 0 1 3 6a2 2 0 0 1 2-2z"/>`,
	"gift":      `<rect x="3" y="9" width="18" height="4"/><rect x="5" y="13" width="14" height="8"/><path d="M12 9v12M12 9c-2-4-6-3-5 0M12 9c2-4 6-3 5 0"/>`,
	"contact":   `<rect x="3" y="5" width="18" height="14" rx="2"/><circle cx="9" cy="11" r="2"/><path d="M6 16c0-2 2-3 3-3s3 1 3 3M14 10h4M14 14h4"/>`,
	"star":      `<path d="m12 3 2.8 5.8 6.2.9-4.5 4.4 1 6.2-5.5-3-5.5 3 1-6.2L3 9.7l6.2-.9z"/>`,
	"tag":       `<path d="M3 12V4h8l10 10-8 8z"/><circle cx="7.5" cy="8.5" r="1.2"/>`,
	"cake":      `<path d="M4 20h16v-7H4zM4 16c3 2 5-2 8 0s5-2 8 0M12 13V8M12 5v1"/>`,
	"trophy":    `<path d="M8 4h8v6a4 4 0 0 1-8 0zM8 6H4v2a3 3 0 0 0 4 3M16 6h4v2a3 3 0 0 1-4 3M12 14v4M8 21h8"/>`,
	"coffee":    `<path d="M4 8h13v6a5 5 0 0 1-5 5H9a5 5 0 0 1-5-5zM17 10h2a2 2 0 0 1 0 5h-2M7 3v2M11 3v2"/>`,
	"takeaway":  `<path d="M5 8h14l-1.5 12h-11zM5 8l2-4h10l2 4M9 12v4M15 12v4"/>`,
	"building":  `<rect x="5" y="3" width="14" height="18" rx="1"/><path d="M9 7h2M13 7h2M9 11h2M13 11h2M9 15h2M13 15h2M10 21v-3h4v3"/>`,
	"utensils":  `<path d="M6 3v8M4 3v5a2 2 0 0 0 4 0V3M6 11v10M17 3c-2 2-3 5-3 8h3v10"/>`,
	"target":    `<circle cx="12" cy="12" r="9"/><circle cx="12" cy="12" r="5"/><circle cx="12" cy="12" r="1"/>`,
	"search":    `<circle cx="11" cy="11" r="6"/><path d="m20 20-4-4"/>`,
	"mail":      `<rect x="3" y="5" width="18" height="14" rx="2"/><path d="m3 7 9 6 9-6"/>`,
	"moon":      `<path d="M20 14a8 8 0 1 1-9-10 6 6 0 0 0 9 10z"/>`,
	"ticket":    `<path d="M3 8a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v2a2 2 0 0 0 0 4v2a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-2a2 2 0 0 0 0-4z"/><path d="M14 6v12"/>`,
	"grad":      `<path d="m2 9 10-5 10 5-10 5z"/><path d="M6 11v5c0 1.5 3 3 6 3s6-1.5 6-3v-5"/>`,
	"wrench":    `<path d="M14 6a4 4 0 0 0 5 5l-9 9a2 2 0 0 1-3-3z"/>`,
	"refresh":   `<path d="M20 12a8 8 0 0 1-14 5M4 12a8 8 0 0 1 14-5M18 3v4h-4M6 21v-4h4"/>`,
	"check":     `<path d="m5 12 5 5 9-10"/>`,
	"arrow":     `<path d="M5 12h14M13 6l6 6-6 6"/>`,
	"layers":    `<path d="m12 3 9 5-9 5-9-5z"/><path d="m3 13 9 5 9-5"/>`,
	"menu":      `<path d="M4 7h16M4 12h16M4 17h16"/>`,
	"chat":      `<path d="M21 12a8 8 0 0 1-11.9 7L4 20l1.1-4.6A8 8 0 1 1 21 12z"/><path d="M9 10h6M9 13h4"/>`,
	"map":       `<path d="M9 4 3 6v14l6-2 6 2 6-2V4l-6 2z"/><path d="M9 4v14M15 6v14"/>`,
	"dot":       `<circle cx="12" cy="12" r="3"/>`,
}

var emojiIcon = map[string]string{
	"🧾": "receipt", "⚡": "bolt", "🧮": "calc", "💳": "card", "🖨": "printer", "👤": "user", "⌨": "keyboard",
	"👨‍🍳": "flame", "🧑‍🍳": "flame", "🍳": "flame", "📺": "screen", "🔔": "bell", "🎨": "sparkle", "🔊": "volume",
	"📱": "phone", "📲": "phone", "🪑": "table", "🛡": "shield", "👀": "eye", "📦": "box", "📉": "trend",
	"⚠": "alert", "🚚": "truck", "🗑": "trash", "🏬": "store", "📋": "clipboard", "📒": "book", "⚙": "cog",
	"📊": "chart", "💸": "wallet", "🔒": "lock", "📤": "upload", "👥": "users", "⏱": "clock", "🌴": "calendar",
	"💼": "briefcase", "🔑": "key", "📍": "pin", "📑": "doc", "🛵": "bike", "📞": "call", "🧑‍✈": "user",
	"💵": "wallet", "💰": "wallet", "📈": "chart", "🎁": "gift", "📇": "contact", "⭐": "star", "🏷": "tag",
	"🎂": "cake", "🧁": "cake", "📅": "calendar", "🏆": "trophy", "⬇": "download", "🍽": "utensils",
	"🍴": "utensils", "☕": "coffee", "🥡": "takeaway", "🏢": "building", "🔐": "lock", "🔍": "search",
	"🎯": "target", "✉": "mail", "🌗": "moon", "🎫": "ticket", "🎉": "sparkle", "🎓": "grad", "🛠": "wrench", "🔄": "refresh",
}

// ico renders an icon from a name or from an old emoji key.
func ico(s string) template.HTML {
	key := strings.NewReplacer("️", "").Replace(strings.TrimSpace(s))
	name := key
	if n, ok := emojiIcon[key]; ok {
		name = n
	}
	p, ok := iconPaths[name]
	if !ok {
		p = iconPaths["dot"]
	}
	return template.HTML(`<svg class="i" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">` + p + `</svg>`)
}
