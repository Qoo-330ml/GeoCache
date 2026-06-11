# Organizer Failed Records API

## Goal

Add one POST endpoint to the membership verification server so Qmby can submit organizer failure records.

Qmby keeps organizer failure records locally. When the user clicks `提交失败记录`, Qmby sends all pending records to the verification server. If the server returns success, Qmby clears the local pending file. If the server returns an error, Qmby keeps the local file for retry.

## Endpoint

```http
POST /api/organizer/failed-records
Content-Type: application/json
Authorization: Bearer {cloud_api_key}
X-License-Key: {cloud_api_key}
```

Use the same API key validation behavior as the existing license verification endpoint.

## Request Body

```json
{
  "email": "user@example.com",
  "beijing_time": "2026-06-04 15:30:00",
  "instance_id": "qmby-0123456789abcdef0123456789abcdef",
  "qmby_version": "0.0.28-fix1",
  "records": "2026-06-04T15:29:10+08:00\tfailed\tMovie.mkv\t/video/source/Movie.mkv\tTMDB search failed\n2026-06-04T15:29:40+08:00\tmanual_reorganize\tShow.S01E01.mkv\t/video/source/Show.S01E01.mkv\t手动重新整理"
}
```

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `email` | string | yes | Member email from the signed license returned by activation-code verification. |
| `beijing_time` | string | yes | Client-side Beijing time, format `YYYY-MM-DD HH:mm:ss`. |
| `instance_id` | string | yes | Stable Qmby instance ID. |
| `qmby_version` | string | no | Qmby version string. |
| `records` | string | yes | Multi-line text. Each line is one organizer failure or manual reorganize record. |

## `records` Format

`records` is a newline-separated text block. Each line has 5 tab-separated fields:

```text
time<TAB>kind<TAB>original_name<TAB>original_path<TAB>message
```

| Column | Example | Description |
| --- | --- | --- |
| `time` | `2026-06-04T15:29:10+08:00` | Client-side RFC3339 timestamp when Qmby recorded the event. |
| `kind` | `failed` | Event type. See allowed values below. |
| `original_name` | `Movie.mkv` | Original file name before organizer processing. |
| `original_path` | `/video/source/Movie.mkv` | Original file path before organizer processing. |
| `message` | `TMDB search failed` | Failure reason or manual reorganize note. |

Allowed `kind` values:

| Kind | Meaning |
| --- | --- |
| `failed` | Automatic organizer failed. |
| `manual_failed` | Manual reorganize failed. |
| `manual_reorganize` | Manual reorganize succeeded; Qmby still records the original file name as requested. |

## Response

Successful response:

```json
{
  "success": true,
  "message": "整理失败记录已提交",
  "count": 2
}
```

Error response:

```json
{
  "success": false,
  "error": "invalid api key"
}
```

Recommended status codes:

| Status | When |
| --- | --- |
| `200` | Accepted and persisted. |
| `400` | Missing or invalid request body. |
| `401` | Missing or invalid API key. |
| `500` | Server-side persistence failure. |

## Suggested Server Behavior

1. Validate API key using the same logic as `/api/license/verify`.
2. Validate `email`, `instance_id`, and non-empty `records`.
3. Split `records` by newline, then split each line by tab.
4. Persist raw `records` and parsed rows if convenient.
5. Return `count` as the number of non-empty record lines accepted.

Do not require every line to be perfectly parseable if the raw text can be stored. Prefer accepting the submission and storing malformed lines with an error marker, because Qmby clears local pending records after a successful response.
