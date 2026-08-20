# AGENTS.md – EasyPlan (Simple Cash Planner)

Diese Datei richtet sich an KI‑Agenten (z. B. Kilo / minimax) und an menschliche Mitwirkende. Sie beschreibt den Projektkontext, die verbindlichen Workflows und die Befehle, die zum Bauen, Testen und Deployen nötig sind.

## Projektüberblick

- **Zweck**: Self‑hosted Webanwendung zur Abschätzung wiederkehrender und einmaliger Ausgaben. Kein Tracking von Einnahmen oder Kontoständen.
- **Zielgruppe**: Privatpersonen, die monatlich, wöchentlich und täglich wissen wollen, wie viel Geld auf welchem Konto (Lifestyle, Sparen, Steuern, …) vorhanden sein muss.
- **Aktueller Status**: Konzeptphase. Code, Tests und Infrastruktur werden Sprint‑weise gemäss `TASKS.md` aufgebaut.

## Tech‑Stack

| Bereich | Auswahl |
|---------|---------|
| Sprache | Go 1.22+ |
| Web‑Framework | `net/http` + `html/template` + HTMX |
| Datenbank | SQLite (WAL, Foreign Keys) |
| Migrations | Eigener Runner (`internal/migrate`) |
| Container | Multi‑Stage `Dockerfile` (golang → distroless/static) |
| Compose | `docker-compose.yml` (Produktion, Port `127.0.0.1:8080`), `docker-compose.dev.yml` (`air` Live‑Reload) |
| CI | GitHub Actions (Lint, Test, Build) |
| Währung | CHF (single currency) |
| Zeitzone | `Europe/Zurich` |
| Auth | V1: keine eingebaute Auth; Reverse‑Proxy‑Auth (Caddy, Traefik, Nginx) |

## Repository‑Layout (geplant)

```
.
├── cmd/server/             # main.go
├── internal/
│   ├── domain/             # structs, validation
│   ├── repo/               # SQLite repositories
│   ├── service/            # business logic (pure)
│   ├── recurrence/         # recurrence generator
│   ├── notify/             # notification dispatcher & targets
│   ├── web/                # HTTP handlers & middleware
│   └── migrate/            # migration runner
├── web/
│   ├── templates/          # html/template files
│   └── static/             # css, js, manifest
├── migrations/             # *.up.sql / *.down.sql
├── docs/                   # zusätzliche Dokumentation
├── scripts/                # smoke.sh, changelog.sh
├── Dockerfile
├── docker-compose.yml
├── docker-compose.dev.yml
├── Konzept.md              # Vision, Architektur, Roadmap
├── CODE_OF_CONDUCT.md      # Verhaltenskodex inkl. Git‑Workflow
├── TASKS.md                # Sprint‑ & Task‑Rundown
└── AGENTS.md               # diese Datei
```

## Verbindliche Workflows

### Git‑Workflow

Der vollständige Workflow ist im globalen Skill `git-workflow` definiert (`~/.agent-os/skills/git-workflow/SKILL.md`). Kurzfassung:

- Direkte Commits auf `main`, `sprint/*`, `release/*`, `epic/*` sind **verboten**.
- Jeder Task liegt auf einem Sub‑Branch (`feat/`, `fix/`, `docs/`, `refactor/`, `chore/`, `data/`).
- Merge mit `--no-ff` in die jeweilige **Sprint‑Branch** (`sprint/<name>`).
- Nach Sprint‑Abschluss: Pull Request von Sprint‑Branch auf `main` mit `--no-ff`‑Merge.
- Sub‑Branches werden nach erfolgreichem Merge gelöscht.
- Commit‑Messages folgen Conventional Commits (`<type>(<scope>): <subject>`), Subject deutsch, ≤ 72 Zeichen.

### Code‑Konventionen

- `gofmt`, `go vet ./...`, `golangci-lint run` ohne Befund.
- Geldwerte ausschliesslich als `int64` in Minor Units (Rappen) speichern.
- Keine `float64` für Geld.
- Keine Secrets im Repository.
- Tests für jede neue Funktion (Unit + ggf. Integration). Tabellarische Edge‑Case‑Testsammlungen für komplexe Logik (z. B. Wiederholungen).
- Public APIs rückwärtskompatibel, ausser bei Major‑Release.

## Befehle

### Entwicklung (lokal)

```bash
# Dev‑Stack starten (Live‑Reload)
docker compose -f docker-compose.dev.yml up

# Lokal ohne Docker
go run ./cmd/server
```

### Bauen & Testen

```bash
go vet ./...
go test ./...
go build -o bin/cashplanner ./cmd/server
```

### Container

```bash
docker build -t cashplanner:dev .
docker compose up -d
```

### Migrations

```bash
go run ./cmd/migrate up     # anwenden
go run ./cmd/migrate down   # zurückrollen (letzte Migration)
```

### Lint & Format

```bash
gofmt -w .
go vet ./...
golangci-lint run
```

## Dokumentation

| Datei | Zweck |
|-------|-------|
| `Konzept.md` | Vision, Architektur, Sprint‑Plan, Sicherheits‑ & Benachrichtigungs‑Konzept |
| `CODE_OF_CONDUCT` | Verhaltenskodex inkl. Git‑Workflow und Sicherheits‑Meldungen |
| `TASKS.md` | Detaillierter Sprint‑ & Task‑Rundown mit Akzeptanzkriterien |
| `AGENTS.md` | Diese Datei – Kontext für KI‑Agenten und neue Mitwirkende |
| `docs/` | Ergänzende Anleitungen (Reverse‑Proxy, Backup, Notifications) |

## Wichtige Regeln für KI‑Agenten

1. **Lies zuerst `Konzept.md` und `TASKS.md`, bevor du Code änderst.**
2. **Halte den Git‑Workflow ein** (Skill `git-workflow`).
3. **Keine direkten Commits auf `main`.** Lege immer einen Sub‑Branch an und merge mit `--no-ff`.
4. **Schreibe Tests** für jede neue Funktion.
5. **Dokumentation** (`Konzept.md`, `docs/`) aktualisieren, sobald sich Verhalten, Datenmodell oder API ändern.
6. **Datenschutz**: Keine echten Kontostände, Kontonummern oder personenbezogenen Daten in Issues, Tests oder Beispielen.
7. **Sicherheit**: Sicherheitslücken niemals öffentlich, sondern per E‑Mail an den Maintainer melden (siehe `CODE_OF_CONDUCT`).
8. **Sprache**: Antworten und Commit‑Subjects auf Deutsch, Code‑Kommentare und Identifier auf Englisch.

## Proaktive Verbesserungen & globale Patterns

Dieses Projekt dient **nicht nur** als Endprodukt, sondern auch als **Quelle für wiederverwendbare globale Konfigurationen** (Skills, MCPs, Workflows, Regeln), die aus den hier gemachten Erfahrungen entstehen. Andere Projekte sollen von diesen Erkenntnissen profitieren, ohne sie neu erarbeiten zu müssen.

- Der Agent erkennt generalisierbare Patterns und **schlägt proaktiv** vor, sie als globalen Skill, MCP, Workflow oder Regel anzulegen.
- Konkrete Vorgehensweise und Auslöser siehe `.kilo/workflow/extract-global-patterns.md`.
- Zielpfade für extrahierte Patterns:
  - Globale Skills → `~/.agent-os/skills/<name>/SKILL.md`
  - Globale MCPs → `~/.config/kilo/kilo.json` (global Kilo config; `mcp`-Sektion)
  - Globale Regeln → `~/.agent-os/policies/<name>.md` (global); `.kilo/rules/<name>.md` (lokal)
  - Lokale Workflows → `.kilo/workflow/<name>.md`
- Vorschläge immer **mit Begründung und konkretem Pfad** machen und vor dem Anlegen bestätigen lassen.
- Im Commit‑Body vermerken, wenn ein Pattern aus EasyPlan in einen globalen Skill/MCP/Workflow überführt wurde.

## Definition of Done

Ein Task gilt als erledigt, wenn:

1. Implementierung, Migrationen und Templates vorhanden sind.
2. Unit‑ und ggf. Integrationstests geschrieben und grün sind.
3. `gofmt`, `go vet`, `golangci-lint`, `go test ./...`, `go build ./...` lokal erfolgreich sind.
4. Akzeptanzkriterium aus `TASKS.md` erfüllt und im PR‑Body dokumentiert ist.
5. CI‑Pipeline grün.
6. Review durch Maintainer gemäss `CODE_OF_CONDUCT`.

## Release‑Reihenfolge

| Version | Inhalt |
|---------|--------|
| `v0.1.0` | Sprints 0–3 (Bootstrap, Accounts, Entries, Recurrence) |
| `v0.2.0` | Sprints 4–5 (Perioden, Reserve) |
| `v0.3.0` | Sprints 6–7 (Dashboard, Timeline) |
| `v0.4.0` | Sprint 8 (Backup & Export) |
| `v1.0.0` | Sprint 9 (Hardening & Release) |
| `v1.1.0` | Sprint 10 (Benachrichtigungen) |
| `v1.2.0` | Sprint 11 (Optionale Features) |