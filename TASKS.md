# Sprint‑ & Task‑Rundown – EasyPlan (Simple Cash Planner)

Dieses Dokument bricht die Sprints aus `Konzept.md` in konkrete, überprüfbare Tasks herunter. Jeder Task ist so formuliert, dass er in einem Issue‑Tracker (z. B. GitHub Projects, Plane) abgebildet werden kann.

Legende: ✅ erledigt · ☐ offen · 🚧 in Arbeit

---

## Sprint 0 – Setup

Ziel: Lauffähiges Projektgerüst, lokale Entwicklungsumgebung und CI.

| ID | Task | Akzeptanz­kriterium |
|----|------|--------------------|
| 0‑1 | Repository initialisieren (README, LICENSE, `.editorconfig`, `.gitignore`) | Repo klonbar, README sichtbar |
| 0‑2 | Konzeptdatei `Konzept.md` ins Repo übernehmen | Datei vorhanden, keine TODOs |
| 0‑3 | Multi‑Stage `Dockerfile` (golang → distroless) | `docker build` erzeugt Image < 20 MB |
| 0‑4 | `docker-compose.yml` (Produktion) | `docker compose up -d` startet Container, Port `127.0.0.1:8080` |
| 0‑5 | `docker-compose.dev.yml` mit `air` Live‑Reload | Änderung an `.go` Datei triggert Neustart < 3 s |
| 0‑6 | CI‑Workflow GitHub Actions (build, test, lint) | Pipeline grün bei Push auf `master` |
| 0‑7 | SQLite‑Migrationen (`accounts`, `entries`, `settings`) | Migrationen laufen idempotent beim Start |
| 0‑8 | Issue‑Templates (`bug`, `feature`, `security`) | Templates im `.github/ISSUE_TEMPLATE/` |

---

## Sprint 1 – Domäne

Ziel: Kerndatenmodell und Wiederholungs­logik.

| ID | Task | Akzeptanz­kriterium |
|----|------|--------------------|
| 1‑1 | `accounts` Repository (Create, Read, Update, Delete, List) | Unit‑Tests grün, `sort_order` wird respektiert |
| 1‑2 | `entries` Repository (CRUD, Filter nach `account_id`, `is_active`) | Unit‑Tests grün |
| 1‑3 | `recurrence.go` – Generator für monatlich, alle N Monate, jährlich, wöchentlich, einmalig | Unit‑Tests decken Monatsende‑Regel ab |
| 1‑4 | Edge‑Case‑Tests: 31. Jan → Feb (28/29), Schaltjahr, `ends_on` < `start_date` | Tabelle mit mindestens 12 Szenarien |
| 1‑5 | Service‑Schicht (`internal/service`) mit reinen Funktionen | Keine I/O‑Abhängigkeiten, voll testbar |
| 1‑6 | Validierung der Eingaben (Name nicht leer, `amount_minor` > 0, gültiges `recurrence_unit`) | Fehlermeldungen lokalisiert |

---

## Sprint 2 – Berechnung

Ziel: Pro‑Konto‑ und Gesamt‑Berechnungen für Monat/Woche/Tag sowie Reserve.

| ID | Task | Akzeptanz­kriterium |
|----|------|--------------------|
| 2‑1 | `MonthlyEquivalent(entry)` inkl. anteiliger Jahresbeträge | Korrekte Werte für Jahr/Quartal/Monat/Woche |
| 2‑2 | `PeriodSummary(accountID)` → `(month, week, day int)` | Wochen = Monat / 4,345; Tage = Monat / 30,4375 |
| 2‑3 | `ReserveRequired(accountID, days int)` | Summiert alle Fälligkeiten ≤ `today + days` |
| 2‑4 | Aggregations‑Service pro Konto und gesamt | Tests mit mehreren Konten |
| 2‑5 | Performance‑Test mit 500 Einträgen / 5 Konten | Berechnung < 10 ms |

---

## Sprint 3 – UI Grundgerüst

Ziel: Mobile‑first Oberfläche, Dashboard, Konten‑ und Einträge‑Listen.

| ID | Task | Akzeptanz­kriterium |
|----|------|--------------------|
| 3‑1 | CSS‑Reset + Basis‑Layout (Mobile‑first, max 480 px Grundbreite) | Läuft im Browser, Lighthouse Mobile > 90 |
| 3‑2 | Dashboard‑Ansicht: pro Konto Monat/Woche/Tag, nächste Zahlung, Reserve | Werte stimmen mit Sprint 2 überein |
| 3‑3 | Konten‑Seite (Liste, Anlegen, Bearbeiten, Deaktivieren) | CRUD via HTMX |
| 3‑4 | Einträge‑Seite (Liste, Filter nach Konto, Anlegen, Bearbeiten, Deaktivieren) | Wiederholung sichtbar, Enddatum optional |
| 3‑5 | Einstellungs‑Seite (Zeitzone, Projektions­fenster, Wunsch‑Puffer pro Konto) | Werte persistent in `settings` |
| 3‑6 | Navigation (Top‑Bar, Bottom‑Tab auf Mobile) | Maximal 5 Hauptregionen erreichbar |

---

## Sprint 4 – Timeline & Warnungen

Ziel: Vorausschau und Sicherheits­hinweise.

| ID | Task | Akzeptanz­kriterium |
|----|------|--------------------|
| 4‑1 | `Timeline(from, to)` Service | Liefert sortierte Liste mit Betrag und Konto |
| 4‑2 | Timeline‑Ansicht (Monats­gruppierung) | HTML‑Darstellung gruppiert nach Monat |
| 4‑3 | Warnungs­logik: `ReserveRequired > wish_buffer` | Banner auf Dashboard mit Link zum Konto |
| 4‑4 | Konfigurierbares Reserve‑Fenster (7/30/60/90 Tage) | Dropdown im Dashboard |
| 4‑5 | Tests: mehrere Fälligkeiten am gleichen Tag | Korrekte Summenbildung |

---

## Sprint 5 – Backup & Export

Ziel: Datenmigration und Wiederherstellung.

| ID | Task | Akzeptanz­kriterium |
|----|------|--------------------|
| 5‑1 | SQLite‑Backup via `VACUUM INTO` | Datei kopierbar, öffnet sich extern |
| 5‑2 | UI‑Button „Backup herunterladen“ | Liefert `.db`‑Datei mit Timestamp |
| 5‑3 | JSON‑Export (alle Konten, Einträge, Einstellungen) | Schema dokumentiert in `docs/export.md` |
| 5‑4 | CSV‑Export der Einträge | UTF‑8, Trenner `,`, Header‑Zeile |
| 5‑5 | Restore‑Funktion mit Validierung | SQLite‑Magic‑String geprüft, Bestätigungs­dialog |

---

## Sprint 6 – Hardening & Release

Ziel: Produktionsreife und erste Veröffentlichung.

| ID | Task | Akzeptanz­kriterium |
|----|------|--------------------|
| 6‑1 | Distroless‑Image, Non‑Root, Read‑Only FS | `docker inspect` zeigt `User: nonroot` |
| 6‑2 | Healthcheck‑Endpoint `/health` (inkl. DB‑Ping) | Liefert `200 OK`, Docker HEALTHCHECK grün |
| 6‑3 | Beispielkonfigurationen für Reverse‑Proxy (Caddy, Traefik, Nginx) | Dateien unter `docs/reverse-proxy/` |
| 6‑4 | Installations‑/Update‑/Backup‑Dokumentation | `docs/`‑Verzeichnis vollständig |
| 6‑5 | Release‑Tag `v1.0.0`, GitHub Release Notes | Image unter `ghcr.io/<owner>/cashplanner:1.0.0` |
| 6‑6 | Smoke‑Test‑Skript (`scripts/smoke.sh`) | Erstellt Konto + Eintrag, ruft `/health` |

---

## Sprint 7 – Benachrichtigungen

Ziel: Aktive Erinnerungen über externe Kanäle.

| ID | Task | Akzeptanz­kriterium |
|----|------|--------------------|
| 7‑1 | `internal/notify` Package mit `Notifier`‑Interface | Mock‑Implementierung vorhanden |
| 7‑2 | Target: Gotify (POST `message?token=…`) | Test gegen lokalen Gotify erfolgreich |
| 7‑3 | Target: ntfy.sh (POST an Topic) | Test gegen `ntfy.sh` erfolgreich |
| 7‑4 | Target: Discord Webhook | Test‑Nachricht erscheint im Kanal |
| 7‑5 | Target: Telegram Bot (`sendMessage`) | Test‑Nachricht erscheint im Chat |
| 7‑6 | Target: Generic Webhook (Header‑basiert) | POST mit Body dokumentiert |
| 7‑7 | Event `entry.due_soon` (X Tage) | Wird täglich 09:00 lokal geprüft |
| 7‑8 | Event `reserve.exceeded` und `reserve.critical` | Severity `warning`/`critical` |
| 7‑9 | Wöchentliche Zusammenfassung (`weekly.summary`) | Sonntag 18:00 lokal |
| 7‑10 | Retry‑Strategie (exponentielles Backoff, max 3) | Fehler werden geloggt, kein Spam |
| 7‑11 | Rate‑Limit (max 1 pro Event/Tag) | Doppelte Auslöser werden zusammengefasst |
| 7‑12 | Einstellungs‑Seite: An/Aus, Ziel, URL, Token, Test‑Button | UI‑Tests grün |
| 7‑13 | TLS‑Prüfung (HTTP‑Ziele abgelehnt) | Konfigurations­validierung |
| 7‑14 | Liste der letzten 20 Benachrichtigungen (Status) | Tabelle im UI |
| 7‑15 | Dokumentation: Gotify‑Setup, Discord‑Webhook, Telegram‑Bot | `docs/notifications.md` |
| 7‑16 | Datenschutz: keine Kontostände in Nachrichten | Code‑Review Checkliste |

---

## Sprint 8 – Optional

Ziel: Komfortfunktionen.

| ID | Task | Akzeptanz­kriterium |
|----|------|--------------------|
| 8‑1 | PWA‑Manifest + Service Worker (Offline‑Shell) | Installierbar, sensible Daten nicht gecacht |
| 8‑2 | Dark Mode (CSS‑Variablen, Toggle in Einstellungen) | Persistenz via `settings` |
| 8‑3 | CSV‑Import von Ausgaben‑Listen (Spalten: name, account, amount, start_date, recurrence) | Validierung + Vorschau |
| 8‑4 | API‑Tokens für externe Skripte (`/api/v1/entries`) | Authentifizierung über `Authorization: Bearer` |
| 8‑5 | Optionale CSV‑Exporte nach Konto getrennt | Filter im Export‑UI |
| 8‑6 | Mehrsprachigkeit (de / en) | Sprachdatei `i18n/<lang>.json` |

---

## Workflow

- Jeder Sprint wird auf einem eigenen Feature‑Branch erstellt (`feat/sprint-<n>-*`).
- Nach Review Merge mit `git merge --no-ff` auf `master` (siehe `CODE_OF_CONDUCT`).
- Tasks werden im Issue‑Tracker abgebildet; ein Task gilt als abgeschlossen, wenn:
  1. Code in einem Pull‑Request enthalten ist,
  2. Unit‑Tests vorhanden und grün,
  3. Akzeptanz­kriterium erfüllt und vom Maintainer bestätigt.

## Reihenfolge der Releases

1. **v0.1.0** – Sprints 0–2 (Domäne + Berechnung, internes Dashboard)
2. **v0.2.0** – Sprint 3–4 (UI, Timeline, Warnungen)
3. **v0.3.0** – Sprint 5 (Backup & Export)
4. **v1.0.0** – Sprint 6 (Hardening & Release)
5. **v1.1.0** – Sprint 7 (Benachrichtigungen)
6. **v1.2.0** – Sprint 8 (Optional)

Damit ist jeder Sprint einem konkreten Release zugeordnet und das Projekt bleibt in kleinen, stabilen Schritten lieferbar.