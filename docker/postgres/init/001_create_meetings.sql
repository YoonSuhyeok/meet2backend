-- ─────────────────────────────────────────────────────────────
-- meetings : 미팅 본체
-- ─────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS meetings (
    id               SERIAL       PRIMARY KEY,
    short_id         VARCHAR(10)  NOT NULL UNIQUE,                 -- 공유 URL용 (예: abc1234)
    invite_code      VARCHAR(8)   NOT NULL UNIQUE,                 -- 초대 코드 (예: ABC-1234)
    invite_policy    VARCHAR(16)  NOT NULL DEFAULT 'auto',         -- auto | approval

    title            VARCHAR(50)  NOT NULL,
    description      VARCHAR(200) NOT NULL DEFAULT '',
    location         VARCHAR(100) NOT NULL DEFAULT '',

    -- 후보 날짜 (ISO YYYY-MM-DD), 1~14일
    dates            DATE[]       NOT NULL,

    -- 시간 범위 (HH:mm, 30분 단위, end는 24:00 허용)
    start_time       VARCHAR(5)   NOT NULL,
    end_time         VARCHAR(5)   NOT NULL,

    -- 주최자 (외부 ID, BFF가 X-User-* 헤더로 전달)
    host_id          VARCHAR(64)  NOT NULL,
    host_name        VARCHAR(50)  NOT NULL,

    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT meetings_dates_len_chk
        CHECK (array_length(dates, 1) BETWEEN 1 AND 14),
    CONSTRAINT meetings_start_time_chk
        CHECK (start_time ~ '^([01][0-9]|2[0-3]):[03]0$'),
    CONSTRAINT meetings_end_time_chk
        CHECK (end_time ~ '^(([01][0-9]|2[0-3]):[03]0|24:00)$'),
    CONSTRAINT meetings_invite_policy_chk
        CHECK (invite_policy IN ('auto', 'approval'))
);

CREATE INDEX IF NOT EXISTS meetings_host_id_idx    ON meetings (host_id);
CREATE INDEX IF NOT EXISTS meetings_created_at_idx ON meetings (created_at DESC);

-- ─────────────────────────────────────────────────────────────
-- votes : 참여자 투표 (미팅당 사용자 1행, slots는 배열로 저장)
-- ─────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS votes (
    id          SERIAL       PRIMARY KEY,
    meeting_id  INTEGER      NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    user_id     VARCHAR(64)  NOT NULL,
    user_name   VARCHAR(50)  NOT NULL,

    -- 슬롯 키 형식: "YYYY-MM-DD-HH:mm" (예: "2026-04-20-09:00")
    slots       TEXT[]       NOT NULL DEFAULT '{}',

    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT votes_meeting_user_uniq UNIQUE (meeting_id, user_id)
);

CREATE INDEX IF NOT EXISTS votes_meeting_id_idx ON votes (meeting_id);
CREATE INDEX IF NOT EXISTS votes_user_id_idx    ON votes (user_id);

-- ─────────────────────────────────────────────────────────────
-- meeting_join_requests : 승인형 참가 요청
-- ─────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS meeting_join_requests (
    id                SERIAL       PRIMARY KEY,
    meeting_id        INTEGER      NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    requester_id      VARCHAR(64)  NOT NULL,
    requester_name    VARCHAR(50)  NOT NULL,
    status            VARCHAR(16)  NOT NULL,
    participant_code  VARCHAR(24),
    processed_by      VARCHAR(64),
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT meeting_join_requests_status_chk
        CHECK (status IN ('pending', 'approved', 'rejected'))
);

CREATE INDEX IF NOT EXISTS meeting_join_requests_meeting_id_idx
    ON meeting_join_requests (meeting_id, created_at DESC);

CREATE INDEX IF NOT EXISTS meeting_join_requests_requester_id_idx
    ON meeting_join_requests (requester_id);

-- ─────────────────────────────────────────────────────────────
-- meeting_participants : 참가 코드 및 투표 조회 권한
-- ─────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS meeting_participants (
    id                SERIAL       PRIMARY KEY,
    meeting_id        INTEGER      NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    requester_id      VARCHAR(64)  NOT NULL,
    requester_name    VARCHAR(50)  NOT NULL,
    participant_code  VARCHAR(24)  NOT NULL UNIQUE,
    status            VARCHAR(16)  NOT NULL DEFAULT 'active',
    approved_by       VARCHAR(64),
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT meeting_participants_status_chk
        CHECK (status IN ('active', 'revoked')),
    CONSTRAINT meeting_participants_meeting_requester_uniq
        UNIQUE (meeting_id, requester_id)
);

CREATE INDEX IF NOT EXISTS meeting_participants_meeting_id_idx
    ON meeting_participants (meeting_id);

CREATE INDEX IF NOT EXISTS meeting_participants_requester_id_idx
    ON meeting_participants (requester_id);

-- ─────────────────────────────────────────────────────────────
-- notification_subscriptions : 미팅 단위 푸시 구독
-- ─────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS notification_subscriptions (
    id                               SERIAL       PRIMARY KEY,
    meeting_id                       INTEGER      NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    user_id                          VARCHAR(64)  NOT NULL,
    device_id                        VARCHAR(64)  NOT NULL,
    endpoint                         TEXT         NOT NULL,
    p256dh                           TEXT         NOT NULL,
    auth                             TEXT         NOT NULL,
    is_standalone                    BOOLEAN      NOT NULL DEFAULT FALSE,
    notification_permission_status   VARCHAR(16)  NOT NULL,
    is_active                        BOOLEAN      NOT NULL DEFAULT TRUE,
    endpoint_status                  VARCHAR(32)  NOT NULL DEFAULT 'active',
    registered_at                    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_verified_at                 TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    created_at                       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at                       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT notification_subscriptions_permission_chk
        CHECK (notification_permission_status IN ('granted', 'denied', 'default')),
    CONSTRAINT notification_subscriptions_endpoint_status_chk
        CHECK (endpoint_status IN ('active', 'sending_suppressed', 'invalid')),
    CONSTRAINT notification_subscriptions_uniq
        UNIQUE (meeting_id, user_id, device_id)
);

CREATE INDEX IF NOT EXISTS notification_subscriptions_meeting_active_idx
    ON notification_subscriptions (meeting_id, is_active, endpoint_status);

CREATE INDEX IF NOT EXISTS notification_subscriptions_user_device_idx
    ON notification_subscriptions (user_id, device_id);

-- ─────────────────────────────────────────────────────────────
-- attendance_nudges : 독촉 발송 이력 및 쿼터 제어
-- ─────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS attendance_nudges (
    id                    SERIAL       PRIMARY KEY,
    nudge_id              VARCHAR(32)  NOT NULL UNIQUE,
    meeting_id            INTEGER      NOT NULL REFERENCES meetings(id) ON DELETE CASCADE,
    trigger_type          VARCHAR(16)  NOT NULL,
    requested_by_user_id  VARCHAR(64),
    message_override      VARCHAR(200),
    target_count          INTEGER      NOT NULL,
    queued_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    sent_at               TIMESTAMPTZ,
    status                VARCHAR(16)  NOT NULL DEFAULT 'queued',
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT attendance_nudges_trigger_type_chk
        CHECK (trigger_type IN ('auto', 'manual')),
    CONSTRAINT attendance_nudges_status_chk
        CHECK (status IN ('queued', 'sent', 'failed')),
    CONSTRAINT attendance_nudges_target_count_chk
        CHECK (target_count >= 0)
);

CREATE INDEX IF NOT EXISTS attendance_nudges_meeting_trigger_queued_idx
    ON attendance_nudges (meeting_id, trigger_type, queued_at DESC);
