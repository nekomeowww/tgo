package tgo

type MemberStatus string

const (
	MemberStatusCreator       MemberStatus = "creator"
	MemberStatusAdministrator MemberStatus = "administrator"
	MemberStatusMember        MemberStatus = "member"
	MemberStatusRestricted    MemberStatus = "restricted"
	MemberStatusLeft          MemberStatus = "left"
	MemberStatusKicked        MemberStatus = "kicked"
)

type ChatType string

const (
	ChatTypePrivate    ChatType = "private"
	ChatTypeGroup      ChatType = "group"
	ChatTypeSuperGroup ChatType = "supergroup"
	ChatTypeChannel    ChatType = "channel"
)
