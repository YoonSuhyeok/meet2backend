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

type NotificationSubscription struct {
	bun.BaseModel `bun:"table:notification_subscriptions,alias:ns"`

	ID                           uint32    `bun:"id,pk,autoincrement" json:"id"`
	MeetingId                    uint32    `bun:"meeting_id,notnull" json:"meetingId"`
	UserId                       string    `bun:"user_id,notnull" json:"userId"`
	DeviceId                     string    `bun:"device_id,notnull" json:"deviceId"`
	Endpoint                     string    `bun:"endpoint,notnull" json:"endpoint"`
	P256dh                       string    `bun:"p256dh,notnull" json:"p256dh"`
	Auth                         string    `bun:"auth,notnull" json:"auth"`
	IsStandalone                 bool      `bun:"is_standalone,notnull,default:false" json:"isStandalone"`
	NotificationPermissionStatus string    `bun:"notification_permission_status,notnull" json:"notificationPermissionStatus"`
	IsActive                     bool      `bun:"is_active,notnull,default:true" json:"isActive"`
	EndpointStatus               string    `bun:"endpoint_status,notnull,default:'active'" json:"endpointStatus"`
	RegisteredAt                 time.Time `bun:"registered_at,nullzero,notnull,default:current_timestamp" json:"registeredAt"`
	LastVerifiedAt               time.Time `bun:"last_verified_at,nullzero,notnull,default:current_timestamp" json:"lastVerifiedAt"`
	CreatedAt                    time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt                    time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`
}

type AttendanceNudge struct {
	bun.BaseModel `bun:"table:attendance_nudges,alias:an"`

	ID                uint32     `bun:"id,pk,autoincrement" json:"id"`
	NudgeId           string     `bun:"nudge_id,notnull,unique" json:"nudgeId"`
	MeetingId         uint32     `bun:"meeting_id,notnull" json:"meetingId"`
	TriggerType       string     `bun:"trigger_type,notnull" json:"triggerType"`
	RequestedByUserId string     `bun:"requested_by_user_id,nullzero" json:"requestedByUserId,omitempty"`
	MessageOverride   string     `bun:"message_override,nullzero" json:"messageOverride,omitempty"`
	TargetCount       int        `bun:"target_count,notnull" json:"targetCount"`
	QueuedAt          time.Time  `bun:"queued_at,nullzero,notnull,default:current_timestamp" json:"queuedAt"`
	SentAt            *time.Time `bun:"sent_at,nullzero" json:"sentAt,omitempty"`
	Status            string     `bun:"status,notnull,default:'queued'" json:"status"`
	CreatedAt         time.Time  `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt         time.Time  `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt"`
}
