# 서버 실행 방법
go run cmd/server/main.go

## Web Push 테스트 발송 환경변수

수동 테스트 푸시 발송 API를 사용하려면 아래 환경변수를 설정하세요.

- `VAPID_PUBLIC_KEY`
- `VAPID_PRIVATE_KEY`
- `VAPID_SUBJECT` (예: `mailto:dev@meet2meet.local`)

PowerShell 예시:

```powershell
$env:VAPID_PUBLIC_KEY = "<your-public-key>"
$env:VAPID_PRIVATE_KEY = "<your-private-key>"
$env:VAPID_SUBJECT = "mailto:dev@meet2meet.local"
go run .\cmd\server\main.go
```

## 테스트 발송 API

- Core API: `POST /meetings/:meetingId/push-test-send`
- BFF API: `POST /api/meetings/:meetingId/push-test-send`

## Attendance nudge worker

`POST /meetings/:meetingId/attendance-nudges` 또는 `POST /meetings/:meetingId/attendance` 요청은 `attendance_nudges`에 queued 레코드를 만든 뒤, 서버 프로세스 내부 worker가 이를 주기적으로 읽어 실제 Web Push를 발송합니다.

- worker는 서버 시작 시 자동으로 실행됩니다.
- VAPID 환경변수가 없으면 worker도 실행되지 않고, 수동 독촉 API는 `push sender is not configured` 오류를 반환합니다.
- Web Push 발송이 terminal failure로 판정되면 endpoint는 자동으로 `sending_suppressed` 또는 `invalid` 상태로 전이됩니다.
- push subscription은 이제 미팅별이 아니라 `userId + deviceId` 기준으로 저장되고, 미팅 API에서는 같은 기기 구독을 재사용합니다.

요청 바디 예시:

```json
{
	"title": "테스트 알림",
	"body": "PWA 알림 도착 확인",
	"url": "/meeting/35",
	"tag": "meeting-35-test"
}
```
