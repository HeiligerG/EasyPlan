# Sprint‑ & Task‑Rundown – EasyPlan (Simple Cash Planner)

Dieses Dokument bricht die Sprints aus `Konzept.md` in **atomare, vertikale Feature‑Slices** herunter. Jeder Task ist so formuliert, dass er:

- in sich geschlossen ist (Backend **und** Frontend, falls relevant, inkl. Tests),
- ein eindeutiges Akzeptanzkriterium besitzt,
- kleine, überprüfbare Änderungen liefert (idealerweise < 4 h Aufwand),
- eine klar definierte Abhängigkeit zu vorherigen Tasks hat.

Legende: ✅ erledigt · ☐ offen · 🚧 in Arbeit · 🔗 Abhängigkeit

---

## Sprint 0 – Bootstrap & Tooling

Ziel: Lauffähiges Projektgerüst, lokale Entwicklungsumgebung und CI.

| ID | Task | Akzeptanz­kriterium | Abhängig von |
|----|------|--------------------|--------------|
| 0‑1 | `README.md`, `LICENSE`, `.editorconfig`, `.gitignore` anlegen | Dateien committed, `git status` sauber | – |
| 0‑2 | `go.mod` mit Modulpfad `github.com/<owner>/cashplanner` initialisieren, `go version` dokumentieren | `go build ./...` läuft leer durch | 0‑1 |
| 0‑3 | `.golangci.yml` mit Lints (`govet`, `staticcheck`, `gofmt`, `unused`) einrichten | `golangci-lint run` ohne Befund | 0‑2 |
| 0‑4 | `Dockerfile` (Multi‑Stage golang → distroless/static) | `docker build` liefert Image < 20 MB | 0‑2 |
| 0‑5 | `docker-compose.yml` (Produktion) – Port `127.0.0.1:8080`, Volume `./data:/data` | `docker compose up -d` startet, `curl localhost:8080/health` 200 | 0‑4 |
| 0‑6 | `docker-compose.dev.yml` mit `air` Live‑Reload, gemountetem Quellcode | Änderung an `.go` triggert Neustart < 3 s | 0‑4 |
| 0‑7 | GitHub‑Actions‑Workflow `.github/workflows/ci.yml` (lint, test, build) | Pipeline grün bei Push auf `master` | 0‑3, 0‑2 |
| 0‑8 | Migration‑Runner (`internal/migrate`) mit SQL‑Dateien unter `migrations/` | `go run ./cmd/migrate up` wendet Migrationen idempotent an | 0‑2 |
| 0‑9 | Issue‑Templates (`bug`, `feature`, `security`) | Templates unter `.github/ISSUE_TEMPLATE/` | 0‑1 |
| 0‑10 | `Makefile` mit Targets `build`, `test`, `lint`, `run`, `docker` | `make help` listet Targets | 0‑3, 0‑8 |

---

## Sprint 1 – Account‑Management (vertikal)

Ziel: Benutzer kann Konten anlegen, anzeigen, bearbeiten und deaktivieren – vollständig über die UI.

| ID | Task | Akzeptanz­kriterium | Abhängig von |
|----|------|--------------------|--------------|
| 1‑1 | Migration `001_accounts.sql` (Tabelle `accounts` + Standardkonten `Lifestyle`, `Sparen`, `Steuern`) | Migration up/down ohne Fehler, Seed-Daten sichtbar | 0‑8 |
| 1‑2 | `internal/domain/account.go` – `Account`‑Struct, Konstruktor, `Validate()` | Unit‑Tests grün | 1‑1 |
| 1‑3 | `internal/repo/account.go` – `Create`, `Get`, `List`, `Update`, `Deactivate` (Soft‑Delete) | Unit‑Tests mit SQLite‑In‑Memory | 1‑2 |
| 1‑4 | `internal/service/account.go` – Orchestrierung + Validierung (Name nicht leer, `sort_order ≥ 0`) | Unit‑Tests | 1‑3 |
| 1‑5 | HTTP‑Handler `GET /accounts` (Liste, HTMX‑Fragment) | `curl -H "HX-Request: true"` liefert HTML‑Tabelle | 1‑4 |
| 1‑6 | HTTP‑Handler `GET /accounts/new` (leeres Formular) | Template gerendert | 1‑4 |
| 1‑7 | HTTP‑Handler `POST /accounts` (anlegen, Antwort: aktualisierte Liste) | Integrationstest: 201 + Liste enthält Konto | 1‑5, 1‑6 |
| 1‑8 | HTTP‑Handler `GET /accounts/{id}/edit` (vorausgefülltes Formular) | 404 bei unbekannter ID | 1‑4 |
| 1‑9 | HTTP‑Handler `POST /accounts/{id}` (aktualisieren) | Integrationstest: Werte geändert | 1‑8 |
| 1‑10 | HTTP‑Handler `POST /accounts/{id}/deactivate` (Soft‑Delete) | Konto verschwindet aus Liste, `is_active=false` | 1‑4 |
| 1‑11 | Template `web/templates/accounts/list.html` (Tabelle, HTMX‑Targets) | Rendert Mockdaten | 1‑5 |
| 1‑12 | Template `web/templates/accounts/form.html` (New/Edit) | Felder validieren clientseitig | 1‑6 |
| 1‑13 | Navigation in `web/templates/layout.html` ergänzen (Link „Konten“) | Link sichtbar auf allen Seiten | 0‑10 |
| 1‑14 | End‑to‑End‑Test: Konto anlegen → auflisten → bearbeiten → deaktivieren | Test grün | 1‑7, 1‑9, 1‑10 |
| 1‑15 | Doku `docs/accounts.md` (CRUD‑Beispiele, Fehlercodes) | Datei committed | 1‑14 |

---

## Sprint 2 – Expense‑Entry‑Management (vertikal)

Ziel: Benutzer kann wiederkehrende und einmalige Ausgaben erfassen und pflegen.

| ID | Task | Akzeptanz­kriterium | Abhängig von |
|----|------|--------------------|--------------|
| 2‑1 | Migration `002_entries.sql` (Tabelle `entries` mit FK `account_id`, Indizes) | Migration sauber | 1‑1 |
| 2‑2 | `internal/domain/entry.go` – Struct + `Validate()` (`amount_minor>0`, gültige `recurrence_unit`) | Unit‑Tests | 2‑1 |
| 2‑3 | `internal/repo/entry.go` – `Create`, `Get`, `ListByAccount`, `Update`, `Deactivate` | Unit‑Tests | 2‑2 |
| 2‑4 | `internal/service/entry.go` – Validierung, Konto‑Existenz prüfen | Unit‑Tests | 2‑3, 1‑4 |
| 2‑5 | HTTP‑Handler `GET /entries` (Filter nach Konto, HTMX‑Fragment) | Filter per `?account_id=` | 2‑4 |
| 2‑6 | HTTP‑Handler `GET /entries/new` (Formular, Konten‑Dropdown) | Dropdown aus 1‑11 | 2‑5, 1‑11 |
| 2‑7 | HTTP‑Handler `POST /entries` (anlegen) | Integrationstest | 2‑6 |
| 2‑8 | HTTP‑Handler `GET /entries/{id}/edit` | 404 bei unbekannter ID | 2‑4 |
| 2‑9 | HTTP‑Handler `POST /entries/{id}` (aktualisieren) | Integrationstest | 2‑8 |
| 2‑10 | HTTP‑Handler `POST /entries/{id}/deactivate` | Eintrag verschwindet | 2‑4 |
| 2‑11 | Template `web/templates/entries/list.html` | Rendert Liste, Edit/Delete‑Buttons | 2‑5 |
| 2‑12 | Template `web/templates/entries/form.html` (Felder, `recurrence_unit` Select) | Clientseitige Validierung | 2‑6 |
| 2‑13 | Navigation erweitern (Link „Einträge“) | Sichtbar | 1‑13 |
| 2‑14 | End‑to‑End‑Test: wiederkehrender Eintrag → einmaliger Eintrag → deaktivieren | Test grün | 2‑7, 2‑9, 2‑10 |
| 2‑15 | Doku `docs/entries.md` | Datei committed | 2‑14 |

---

## Sprint 3 – Recurrence‑Engine (vertikal)

Ziel: Aus einer Entry‑Regel die konkreten Fälligkeiten generieren – testbar und performant.

| ID | Task | Akzeptanz­kriterium | Abhängig von |
|----|------|--------------------|--------------|
| 3‑1 | `internal/domain/event.go` – `Event`‑Struct (Datum, Konto, Betrag, Name) | Unit‑Tests | 2‑2 |
| 3‑2 | `internal/recurrence/next.go` – `NextOccurrence(entry, after)` für `once` | Tests | 2‑2 |
| 3‑3 | `NextOccurrence` für `day`/`week` (inkl. Intervall) | Tests | 3‑2 |
| 3‑4 | `NextOccurrence` für `month` mit Intervall 1 (Monatsende‑Regel) | Tests inkl. 31. Jan → Feb | 3‑2 |
| 3‑5 | `NextOccurrence` für `month` mit Intervall > 1 | Tests | 3‑4 |
| 3‑6 | `NextOccurrence` für `year` (Schaltjahr 29. Feb) | Tests | 3‑2 |
| 3‑7 | `GenerateOccurrences(entry, from, to) []Event` (Loop mit `NextOccurrence`) | Tests | 3‑6 |
| 3‑8 | Edge‑Case‑Testsammlung (`tests/recurrence/`) – mind. 20 Szenarien | Tabelle gepflegt | 3‑7 |
| 3‑9 | Benchmark `BenchmarkGenerateOccurrences_500Entries_12Months` | < 10 ms | 3‑7 |
| 3‑10 | Service `internal/service/timeline.go` – `Timeline(from,to)` über alle aktiven Einträge | Tests | 3‑7, 2‑3 |
| 3‑11 | HTTP‑Handler `GET /api/entries/{id}/occurrences?from=&to=` (JSON) | Beispielantwort dokumentiert | 3‑10 |

---

## Sprint 4 – Perioden‑Berechnung (vertikal)

Ziel: Pro Konto und gesamt den monatlichen, wöchentlichen und täglichen Bedarf anzeigen.

| ID | Task | Akzeptanz­kriterium | Abhängig von |
|----|------|--------------------|--------------|
| 4‑1 | `internal/service/period.go` – `MonthlyEquivalent(entry)` | Unit‑Tests (Monat, Quartal, Jahr, Woche) | 2‑2 |
| 4‑2 | `PeriodSummary(accountID)` → `(month, week, day int)` | Tests mit mehreren Konten | 4‑1, 2‑3 |
| 4‑3 | `TotalSummary()` → aggregiert über alle Konten | Tests | 4‑2 |
| 4‑4 | HTTP‑Fragment `GET /accounts/{id}/summary` (HTML) | Integrationstest | 4‑2 |
| 4‑5 | Template `web/templates/accounts/summary.html` (Monat/Woche/Tag) | Rendert | 4‑4 |
| 4‑6 | End‑to‑End‑Test: Seed‑Daten → erwartete Werte → Fragment prüfen | Test grün | 4‑5 |

---

## Sprint 5 – Reserve‑Berechnung (vertikal)

Ziel: Pro Konto die benötigte Reserve für ein wählbares Zeitfenster anzeigen.

| ID | Task | Akzeptanz­kriterium | Abhängig von |
|----|------|--------------------|--------------|
| 5‑1 | `internal/service/reserve.go` – `ReserveRequired(accountID, days int)` | Tests (7/30/60/90 Tage) | 3‑10, 2‑3 |
| 5‑2 | Migration `003_settings.sql` (`reserve_window_days`, `wish_buffer_<account_id>`) | Migration sauber | 1‑1 |
| 5‑3 | `internal/service/settings.go` – `GetInt/SetInt` | Tests | 5‑2 |
| 5‑4 | HTTP‑Fragment `GET /accounts/{id}/reserve?days=` | Integrationstest | 5‑1, 5‑3 |
| 5‑5 | Template `web/templates/accounts/reserve.html` (Betrag, nächste Fälligkeit) | Rendert | 5‑4 |
| 5‑6 | UI‑Toggle im Dashboard (7/30/60/90) | Funktioniert via HTMX | 5‑5 |
| 5‑7 | Warnungs‑Logik: `ReserveRequired > wish_buffer` → Severity `warning` | Test | 5‑1, 5‑3 |

---

## Sprint 6 – Dashboard (vertikal)

Ziel: Mobile‑first Übersicht mit Perioden‑ und Reserve‑Informationen pro Konto.

| ID | Task | Akzeptanz­kriterium | Abhängig von |
|----|------|--------------------|--------------|
| 6‑1 | `web/static/css/base.css` (Reset, Variablen, Mobile‑first) | Lighthouse Mobile > 90 | 0‑10 |
| 6‑2 | Layout‑Template `web/templates/layout.html` (Header, Bottom‑Tabs) | Rendert auf Mobile/Desktop | 6‑1 |
| 6‑3 | Route `GET /` – Dashboard‑Controller | Liefert 200 | 6‑2 |
| 6‑4 | Dashboard‑Komponente „Konto‑Karte“ (Name, Monat/Woche/Tag, nächste Zahlung) | Integrationstest | 4‑5, 6‑3 |
| 6‑5 | Dashboard‑Komponente „Reserve‑Übersicht“ mit Toggle | Integrationstest | 5‑5, 6‑3 |
| 6‑6 | Dashboard‑Komponente „Gesamtsumme“ | Integrationstest | 4‑3, 6‑3 |
| 6‑7 | End‑to‑End‑Test: Seed → Dashboard aufrufen → HTML‑Snapshots vergleichen | Test grün | 6‑4, 6‑5, 6‑6 |

---

## Sprint 7 – Timeline‑Ansicht (vertikal)

Ziel: Chronologische Vorschau der nächsten 12 Monate.

| ID | Task | Akzeptanz­kriterium | Abhängig von |
|----|------|--------------------|--------------|
| 7‑1 | `internal/service/timeline.go` – `GroupByMonth(events)` | Tests | 3‑10 |
| 7‑2 | HTTP‑Route `GET /timeline?months=12` | Integrationstest | 7‑1 |
| 7‑3 | Template `web/templates/timeline.html` (Monats­gruppierung) | Rendert | 7‑2 |
| 7‑4 | Navigation „Timeline“ hinzufügen | Sichtbar | 6‑2 |
| 7‑5 | End‑to‑End‑Test | Test grün | 7‑3 |

---

## Sprint 8 – Backup & Export (vertikal)

Ziel: Daten sicher exportieren und wiederherstellen.

| ID | Task | Akzeptanz­kriterium | Abhängig von |
|----|------|--------------------|--------------|
| 8‑1 | `internal/service/backup.go` – `CreateBackup()` via SQLite `VACUUM INTO` | Erzeugt konsistente `.db` | 0‑8 |
| 8‑2 | HTTP `GET /backup/download` (Datei‑Stream) | Browser lädt `.db` herunter | 8‑1 |
| 8‑3 | UI‑Button „Backup herunterladen“ in Einstellungen | Klick löst Download aus | 8‑2 |
| 8‑4 | `internal/service/export.go` – `ExportJSON()` (alle Tabellen) | Schema dokumentiert | 8‑1 |
| 8‑5 | HTTP `GET /export/json` | Download startet | 8‑4 |
| 8‑6 | `ExportCSV(entries)` | UTF‑8, Header | 2‑3 |
| 8‑7 | HTTP `GET /export/csv` | Download startet | 8‑6 |
| 8‑8 | UI‑Buttons „JSON exportieren“, „CSV exportieren“ | Sichtbar | 8‑5, 8‑7 |
| 8‑9 | `internal/service/restore.go` – `RestoreUpload(file)` (Magic‑Check, atomarer Austausch) | Tests | 8‑1 |
| 8‑10 | HTTP `POST /restore` (multipart, Bestätigungsdialog) | Integrationstest | 8‑9 |
| 8‑11 | UI‑Formular „Wiederherstellen“ mit Bestätigung | Funktioniert | 8‑10 |
| 8‑12 | End‑to‑End‑Test: Backup → Restore in leere DB | Test grün | 8‑2, 8‑11 |

---

## Sprint 9 – Hardening & Release (horizontal, aber klein)

Ziel: Produktionsreife, Sicherheit, Dokumentation, erstes Release.

| ID | Task | Akzeptanz­kriterium | Abhängig von |
|----|------|--------------------|--------------|
| 9‑1 | Container: Non‑Root‑User, `read_only: true` in Compose | `docker inspect` zeigt `User: nonroot` | 0‑5 |
| 9‑2 | `/health`‑Endpoint inkl. DB‑Ping | Liefert `200 OK` bei laufender DB | 0‑5 |
| 9‑3 | Structured Logging (`log/slog`) + Request‑ID‑Middleware | Logs enthalten Request‑ID | 9‑2 |
| 9‑4 | Security‑Header‑Middleware (CSP, HSTS, X‑Content‑Type‑Options) | Header sichtbar | 9‑3 |
| 9‑5 | Rate‑Limit‑Middleware (z. B. `golang.org/x/time/rate`) | 429 bei Überlast | 9‑3 |
| 9‑6 | Beispielkonfigurationen `docs/reverse-proxy/caddy.md`, `traefik.md`, `nginx.md` | Dateien committed | 9‑1 |
| 9‑7 | Installations‑/Update‑/Backup‑Dokumentation | `docs/operations.md` | 8‑12 |
| 9‑8 | `CHANGELOG.md`‑Generator (`scripts/changelog.sh`) | Liest Conventional Commits | 0‑7 |
| 9‑9 | Release‑Tag `v1.0.0`, GitHub‑Release‑Notes | GHCR‑Image `ghcr.io/<owner>/cashplanner:1.0.0` | 9‑8 |
| 9‑10 | Smoke‑Test‑Skript `scripts/smoke.sh` (Container starten, `/health`, Backup) | Skript exit 0 | 9‑2, 9‑9 |

---

## Sprint 10 – Benachrichtigungen (vertikal pro Ziel)

Ziel: Aktive Erinnerungen über externe Kanäle, vollständig konfigurierbar.

| ID | Task | Akzeptanz­kriterium | Abhängig von |
|----|------|--------------------|--------------|
| 10‑1 | `internal/notify/types.go` – `Notification`‑Struct, `Severity` Enum | Unit‑Tests | 3‑10 |
| 10‑2 | `internal/notify/dispatcher.go` – `Notifier`‑Interface, Retry (exp. Backoff, max 3) | Tests mit Mock | 10‑1 |
| 10‑3 | `internal/notify/target_gotify.go` – HTTP‑POST an Gotify | Integrationstest gegen lokalen Gotify | 10‑2 |
| 10‑4 | `internal/notify/target_ntfy.go` – HTTP‑POST an ntfy.sh | Test gegen `ntfy.sh` (sandbox topic) | 10‑2 |
| 10‑5 | `internal/notify/target_discord.go` – Webhook‑POST | Mock‑Test, JSON‑Struktur verifiziert | 10‑2 |
| 10‑6 | `internal/notify/target_telegram.go` – `sendMessage` | Mock‑Test | 10‑2 |
| 10‑7 | `internal/notify/target_generic.go` – beliebiger Webhook (Header + Body) | Test | 10‑2 |
| 10‑8 | `internal/notify/store.go` – Persistenz der letzten 20 Nachrichten (Status, Zeit) | Tests | 10‑1 |
| 10‑9 | `internal/service/notifier.go` – Eventquelle `entry.due_soon` (X Tage) | Test mit Fake‑Clock | 10‑2, 2‑3 |
| 10‑10 | Eventquelle `reserve.exceeded` (warning) und `reserve.critical` (Faktor 1,5) | Test | 10‑9, 5‑7 |
| 10‑11 | Eventquelle `weekly.summary` (Sonntag 18:00 lokal) | Test mit Cron‑Mock | 10‑9 |
| 10‑12 | Migration `004_notification_settings.sql` (`notify_enabled`, `notify_target_*`) | Migration sauber | 5‑2 |
| 10‑13 | UI‑Sektion „Benachrichtigungen“ in Einstellungen (An/Aus, Dropdown, URL, Token, Test‑Button) | Tests | 10‑12 |
| 10‑14 | UI‑Liste „Letzte Benachrichtigungen“ (Status, Zeit, Empfänger) | Test | 10‑8, 10‑13 |
| 10‑15 | Cron‑/Scheduler‑Subsystem (`internal/scheduler`) registriert Events | Test | 10‑9, 10‑10, 10‑11 |
| 10‑16 | Doku `docs/notifications.md` (Gotify‑Setup, Discord‑Webhook, Telegram‑Bot, ntfy) | Datei committed | 10‑3, 10‑4, 10‑5, 10‑6 |
| 10‑17 | Datenschutz‑Checkliste: keine Kontostände in Nachrichten | Review dokumentiert | 10‑13 |

---

## Sprint 11 – Optionale Features (vertikal pro Feature)

Ziel: Komfortfunktionen, jeweils als eigenständiger Slice.

| ID | Task | Akzeptanz­kriterium | Abhängig von |
|----|------|--------------------|--------------|
| 11‑1 | PWA‑Manifest `web/static/manifest.json` + Service Worker (Offline‑Shell) | Installierbar, keine sensiblen Daten gecacht | 6‑2 |
| 11‑2 | Dark‑Mode (CSS‑Variablen, Toggle in Einstellungen) | Persistenz via `settings` | 6‑1 |
| 11‑3 | CSV‑Import (`POST /entries/import`) – Validierung + Vorschau | Integrationstest | 2‑4 |
| 11‑4 | API‑Tokens (`/api/v1/entries`) mit Bearer‑Auth | Dokumentiert, Tests | 2‑4, 9‑5 |
| 11‑5 | Mehrsprachigkeit (de/en) via `i18n/<lang>.json` | Sprachumschaltung in UI | 6‑2 |
| 11‑6 | Erweiterte Suche in Einträgen (Volltext, Betragsbereich) | UI + Service | 2‑4 |
| 11‑7 | Diagramme (Monatsbedarf je Konto) – simple SVG‑Balken | Integrationstest | 4‑2 |

---

## Release‑Reihenfolge

| Release | Enthält Sprints |
|---------|-----------------|
| **v0.1.0** | Sprint 0, 1, 2, 3 |
| **v0.2.0** | Sprint 4, 5 |
| **v0.3.0** | Sprint 6, 7 |
| **v0.4.0** | Sprint 8 |
| **v1.0.0** | Sprint 9 |
| **v1.1.0** | Sprint 10 |
| **v1.2.0** | Sprint 11 (jeweils einzeln) |

## Definition of Done (für jeden Task)

1. Implementierung vorhanden (Code, Templates, Migrationen).
2. Unit‑Test (oder Integrationstest, falls UI) geschrieben und grün.
3. Akzeptanzkriterium erfüllt und im PR beschrieben.
4. `make lint test build` lokal erfolgreich.
5. CI‑Pipeline grün.
6. Dokumentation (falls Sicht ändert) aktualisiert.
7. Review durch Maintainer (mind. ein Approval) gemäß `CODE_OF_CONDUCT`.

## Workflow

- Jeder Task erhält ein eigenes Issue im Tracker (Plane/GitHub Projects).
- Branch‑Naming: `feat/<sprint>-<task-id>-<kurzbeschreibung>`.
- Merge auf `master` mit `git merge --no-ff` gemäß `CODE_OF_CONDUCT`.
- Keine Commits direkt auf `master`.