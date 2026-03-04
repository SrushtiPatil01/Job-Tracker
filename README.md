# Job-Tracker

# Job Tracker

Job application tracking backend with Kafka-based reminder pipeline, Redis idempotency, and Prometheus observability.

Built With: Go · goroutines · Apache Kafka ·  Redis ·  PostgreSQL ·  AWS S3 ·  Prometheus ·  Docker

---

## What This Project Does

You log every job application through a REST API. Seven days after logging, if the status is still "Applied," you automatically get a reminder email via SendGrid. A stats endpoint shows your pipeline — total applications, response rate, breakdown by stage.

When an application is created, an event is published to Kafka. A consumer reads these events, checks if a reminder is due, and sends the email. File attachments (job descriptions) are uploaded directly to AWS S3 via pre-signed URLs — the file never passes through the API server.

The hard problems this solves:

- **Duplicate reminders** — Kafka guarantees at-least-once delivery. If the consumer crashes and replays events, the same reminder could send twice. Job Tracker stores a Redis key per application per week (`reminder_sent:{applicationId}:{isoWeek}`) with a 7-day TTL. If the key exists, the email is skipped.
- **Slow sequential processing** — The reminder consumer needs to check hundreds or thousands of applications, each involving a database read and potentially a network call. Job Tracker runs one goroutine per pending application, processing all of them in parallel. 10K reminder events processed with Kafka consumer lag under 200ms.
- **File upload bottleneck** — Instead of routing files through the API server (memory pressure, slow uploads), Job Tracker generates a pre-signed S3 URL valid for 10 minutes. The client uploads directly to S3. The API stores only the S3 object key.
- **No visibility into system health** — Prometheus metrics track `reminders_sent_total`, `reminders_failed_total`, and `kafka_consumer_lag_seconds`. If SendGrid starts failing or the consumer falls behind, you see it immediately.

---
## Tech Stack

| Layer | Technology |
|---|---|
| **Language** | Go |
| **Backend** | Gin / net/http |
| **Database** | PostgreSQL  |
| **Messaging** | Apache Kafka  |
| **Caching** | Redis |
| **Storage** | AWS S3  |
| **Email** | SendGrid API  |
| **Observability** | Prometheus  |
| **Infrastructure** | Docker Compose  |

---

## Architecture

<img width="2470" height="1080" alt="image" src="https://github.com/user-attachments/assets/765bcca7-267b-4ed8-bcc3-b10ca5dd70d4" />


**How the pieces connect:**

| Flow | Path |
|------|------|
| **Log Application** | User → API → save to DB → publish event to Kafka → return `201` |
| **Send Reminder** | Kafka Consumer → check if 7+ days old → check Redis key → send email via SendGrid → set Redis key |
| **Upload Attachment** | User → API → generate pre-signed S3 URL → user uploads directly to S3 |
| **View Stats** | User → API → query DB → return response rate, stage breakdown |
| **Metrics** | Prometheus → scrape `/metrics` every 15s → track reminders sent, failed, consumer lag |

---

## Application Status Flow

```
APPLIED ──────▶ OA ──────▶ INTERVIEW ──────▶ OFFER
   │                                           
   │            (any status)                   
   └──────────────────────────────────────▶ REJECTED

Reminder fires only when status has been APPLIED for 7+ days.
Updating status to anything else stops the reminder.
```

---

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/applications` | Log a new job application |
| `PATCH` | `/applications/:id` | Update application status |
| `GET` | `/applications` | List all applications, filterable by status and date |
| `GET` | `/applications/stats` | Response rate and stage breakdown |
| `POST` | `/applications/:id/attachment` | Get pre-signed S3 URL for uploading job description PDF |
| `GET` | `/metrics` | Prometheus metrics endpoint |

