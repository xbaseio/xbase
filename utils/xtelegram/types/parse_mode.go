package types

type ParseMode string

const (
	ModeNone            ParseMode = ""
	ParseModeMarkdownV1 ParseMode = "Markdown"
	ParseModeMarkdown   ParseMode = "MarkdownV2"
	ParseModeHTML       ParseMode = "HTML"
)

func (pm ParseMode) String() string {
	return string(pm)
}
