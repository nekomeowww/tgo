package tgo

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/nekomeowww/xo"
)

var (
	escapeForMarkdownV2MarkdownLinkRegexp = regexp.MustCompile(`(\[[^\][]*]\(http[^()]*\))|[_*[\]()~>#+=|{}.!-]`)
)

// EscapeForMarkdownV2
//
// 1. 任何字符码表在 1 到 126 之间的字符都可以加前缀 '\' 字符来转义，在这种情况下，它被视为一个普通字符，而不是标记的一部分。这意味着 '\' 字符通常必须加前缀 '\' 字符来转义。
// 2. 在 pre 和 code 的实体中，所有 '`' 和 '\' 字符都必须加前缀 '\' 字符转义。
// 3. 在所有其他地方，字符 '_', '*', '[', ']', '(', ')', '~', '`', '>', '#', '+', '-', '=', '|', '{', '}', '.', '!' 必须加前缀字符 '\' 转义。
//
// https://core.telegram.org/bots/api#formatting-options
func EscapeStringForMarkdownV2(src string) string {
	var result string

	escapingIndexes := make([][]int, 0)

	// 查询需要转义的字符
	for _, match := range escapeForMarkdownV2MarkdownLinkRegexp.FindAllStringSubmatchIndex(src, -1) {
		if match[2] != -1 && match[3] != -1 { // 匹配到了 markdown 链接
			continue // 不需要转义
		}

		escapingIndexes = append(escapingIndexes, match) // 需要转义
	}

	// 对需要转义的字符进行转义
	var lastMatchedIndex int

	for i, match := range escapingIndexes {
		if i == 0 {
			result += src[lastMatchedIndex:match[0]]
		} else {
			result += src[escapingIndexes[i-1][1]:match[0]]
		}

		result += `\` + src[match[0]:match[1]]
		lastMatchedIndex = match[1]
	}
	if lastMatchedIndex < len(src) {
		result += src[lastMatchedIndex:]
	}

	return result
}

func FullNameFromFirstAndLastName(firstName, lastName string) string {
	if lastName == "" {
		return firstName
	}
	if firstName == "" {
		return lastName
	}
	if xo.ContainsCJKChar(firstName) && !xo.ContainsCJKChar(lastName) {
		return firstName + " " + lastName
	}
	if !xo.ContainsCJKChar(firstName) && xo.ContainsCJKChar(lastName) {
		return lastName + " " + firstName
	}
	if xo.ContainsCJKChar(firstName) && xo.ContainsCJKChar(lastName) {
		return lastName + " " + firstName
	}

	return firstName + " " + lastName
}

// EscapeHTMLSymbols
//
//	< with &lt;
//	> with &gt;
//	& with &amp;
func EscapeHTMLSymbols(str string) string {
	str = strings.ReplaceAll(str, "<", "&lt;")
	str = strings.ReplaceAll(str, ">", "&gt;")
	str = strings.ReplaceAll(str, "&", "&amp;")

	return str
}

var regexpHTMLBlocks = regexp.MustCompile(`<[^>]*>`)

// RemoveHTMLBlocksFromString
//
//	<any> with ""
//	</any> with ""
func RemoveHTMLBlocksFromString(str string) string {
	str = regexpHTMLBlocks.ReplaceAllString(str, "")

	return str
}

var (
	matchMdTitles = regexp.MustCompile(`(?m)^(#){1,6} (.)*(\n)?`)
)

func ReplaceMarkdownTitlesToTelegramBoldElement(text string) string {
	return matchMdTitles.ReplaceAllStringFunc(text, func(s string) string {
		// remove hashtag
		for strings.HasPrefix(s, "#") {
			s = strings.TrimPrefix(s, "#")
		}
		// remove space
		s = strings.TrimPrefix(s, " ")

		sRunes := []rune(s)
		ret := "<b>" + string(sRunes[:len(sRunes)-1])

		// if the line ends with a newline, add a newline to the end of the bold element
		if strings.HasSuffix(s, "\n") {
			return ret + "</b>\n"
		}

		// otherwise, just return the bold element
		return ret + string(sRunes[len(sRunes)-1]) + "</b>"
	})
}

func MapChatTypeToChineseText(chatType ChatType) string {
	switch chatType {
	case ChatTypePrivate:
		return "私聊"
	case ChatTypeGroup:
		return "群组"
	case ChatTypeSuperGroup:
		return "超级群组"
	case ChatTypeChannel:
		return "频道"
	default:
		return "未知"
	}
}

func MapMemberStatusToChineseText(memberStatus MemberStatus) string {
	switch memberStatus {
	case MemberStatusCreator:
		return "创建者"
	case MemberStatusAdministrator:
		return "管理员"
	case MemberStatusMember:
		return "成员"
	case MemberStatusRestricted:
		return "受限成员"
	case MemberStatusLeft:
		return "已离开"
	case MemberStatusKicked:
		return "已被踢出"
	default:
		return "未知"
	}
}

func FormatFullNameAndUsername(fullName, username string) string {
	if utf8.RuneCountInString(fullName) >= 10 && username != "" {
		return username
	}

	return strings.ReplaceAll(fullName, "#", "")
}

func FormatChatID(chatID int64) string {
	chatIDStr := strconv.FormatInt(chatID, 10)
	if strings.HasPrefix(chatIDStr, "-100") {
		return strings.TrimPrefix(chatIDStr, "-100")
	}

	return chatIDStr
}
