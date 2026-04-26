package model

import (
	"time"

	"github.com/uptrace/bun"
)

type Meeting struct {
	bun.BaseModel `bun:"table:meetings,alias:m"`

	ID           uint32 `bun:"id,pk,autoincrement" json:"id"`
	ShortId      string `bun:"short_id,unique,notnull" json:"shortId"`
	InviteCode   string `bun:"invite_code,unique,notnull" json:"inviteCode"`
	InvitePolicy string `bun:"invite_policy,notnull,default:'auto'" json:"invitePolicy"`

	Title       string `bun:"title,notnull" json:"title"`
	Description string `bun:"description" json:"description"`
	Location    string `bun:"location" json:"location"`

	Dates     []time.Time `bun:"dates,array" json:"dates"`
	StartTime string      `bun:"start_time" json:"startTime"`
	EndTime   string      `bun:"end_time" json:"endTime"`

	HostId   string `bun:"host_id,notnull" json:"hostId"`
	HostName string `bun:"host_name,notnull" json:"hostName"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`
}

type Vote struct {
	bun.BaseModel `bun:"table:votes,alias:v"`
	ID            uint32   `bun:"id,pk,autoincrement" json:"id"`
	MeetingId     uint32   `bun:"meeting_id,notnull" json:"meetingId"`
	UserId        string   `bun:"user_id,notnull" json:"userId"`
	UserName      string   `bun:"user_name,notnull" json:"userName"`
	Slots         []string `bun:"slots,array" json:"slots"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`
}

type JoinRequest struct {
	bun.BaseModel `bun:"table:meeting_join_requests,alias:mjr"`

	ID              uint32 `bun:"id,pk,autoincrement" json:"id"`
	MeetingId       uint32 `bun:"meeting_id,notnull" json:"meetingId"`
	RequesterId     string `bun:"requester_id,notnull" json:"requesterId"`
	RequesterName   string `bun:"requester_name,notnull" json:"requesterName"`
	Status          string `bun:"status,notnull" json:"status"`
	ParticipantCode string `bun:"participant_code,nullzero" json:"participantCode,omitempty"`
	ProcessedBy     string `bun:"processed_by,nullzero" json:"processedBy,omitempty"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`
}

type MeetingParticipant struct {
	bun.BaseModel `bun:"table:meeting_participants,alias:mp"`

	ID              uint32 `bun:"id,pk,autoincrement" json:"id"`
	MeetingId       uint32 `bun:"meeting_id,notnull" json:"meetingId"`
	RequesterId     string `bun:"requester_id,notnull" json:"requesterId"`
	RequesterName   string `bun:"requester_name,notnull" json:"requesterName"`
	ParticipantCode string `bun:"participant_code,notnull" json:"participantCode"`
	Status          string `bun:"status,notnull" json:"status"`
	ApprovedBy      string `bun:"approved_by,nullzero" json:"approvedBy,omitempty"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`
}

type MeetingWithVotes struct {
	Meeting *Meeting `json:"meeting"`
	Votes   []*Vote  `json:"votes"`
}
