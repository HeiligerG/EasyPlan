# Übergabe: Phase 3 – Betrieb

Erstellt: 2026-10-31 · Gehört zu `.kilo/plans/2026-10-30-easyplan-v1.md` (Tasks T16–T22)

Diese Datei ist für eine neue planner-Session gedacht. Subagenten und die neue Session kennen den bisherigen Chat nicht.

## Startbedingung

Phase 2 ist abgeschlossen, wenn der PR `feat/phase-2-ui` in `master` gemergt ist. Erst dann:

1. `git checkout master` und `git pull` (der Mensch führt Befehle aus, die der planner nicht darf, siehe unten).
2. Branch `feat/phase-3-betrieb` von `master` anlegen.
3. Plan-Status in `2026-10-30-easyplan-v1.md` lesen (Phase 2 = done), T16 starten.

## Stand nach Phase 2

- `internal/plan` (rein), `internal/store` (SQLite, Migration, CRUD), `internal/web` (Dashboard, Einträge, Konten, Timeline, Origin-Prüfung, CSP, Geldformat), `cmd/easyplan/main.go` (Server, TZ, Shutdown, Timeouts).
- Qualität zuletzt: `gofmt` leer, `go vet` grün, `go test ./...` (auch `-race`, `CGO_ENABLED=0`) grün, 287 Top-Level-Tests, Coverage web 94.4 %, cmd 77.1 %, `mkdocs build --strict` Exit 0.
- Der Subcommand `healthcheck` ist noch ein Platzhalter (liefert Fehler). `/health` und Backup gibt es nicht.
- Konfiguration: `EASYPLAN_ADDR` (Default `:8080`), `EASYPLAN_DATA` (Default `/data`, Datei `easyplan.db`), `TZ` (Default `Europe/Zurich`).

## Entscheide, die Phase 3 betreffen

- Schreibende Requests brauchen einen `Origin`-Header, der zu `Host` passt (sonst 403). Ein Reverse Proxy muss `Host` erhalten.
- Die Antwort auf falsche Methoden ist 404 statt 405 (Catch-all `/`). Neue Routen (`/health`, Backup) beachten das; `/health` als `GET /health` registrieren, vor dem Catch-all ist kein Problem, da der Mux das spezifischere Muster wählt.
- CSP `default-src 'self'`: kein Inline-Script, kein `style=`-Attribut.
- Backup: `VACUUM INTO` darf bei `read_only: true` nicht nach `/tmp` schreiben. Temporäre Datei in `/data` anlegen, nach dem Download löschen. Im Store fehlt dafür `VacuumInto`. Das Backup liefert die komplette Datenbank, deshalb sollte der Download als `POST` laufen (Origin-Schutz), nicht als `GET`; mit der Begründung dokumentieren.
- Healthcheck in distroless: Exec-Form `HEALTHCHECK CMD ["/easyplan","healthcheck"]`.
- Volume-Rechte: `./data` muss dem Nutzer `nonroot` (UID 65532) gehören, in der README vermerken.
- `time/tzdata` ist bereits in `main.go` eingebettet; im Image gegen `TZ=Europe/Zurich` verifizieren.
- Bind-Adresse: `EASYPLAN_ADDR=:8080` bindet ausserhalb Docker an alle Schnittstellen, ohne Authentifizierung. Compose bindet `127.0.0.1:8080:8080`. In der README deutlich machen.
- `serve` setzt einmal `time.Local`; im Betrieb unkritisch, nicht nebenläufig aufrufbar (Test läuft deshalb in eigenem Prozess).
- HTMX 2.0.11 ist eingebettet, aber ungenutzt. Der Mensch hat keine Entfernung verlangt.
- `mkdocs.yml` hat kein `site_url` (leere Sitemap). Entscheiden, falls die Doku publiziert wird.

## Arbeitsumgebung (Erfahrungen)

- Go liegt unter `~/sdk/go` (Version 1.27.3); wenn `go` nicht im PATH ist, Pfad ergänzen.
- Docs-Build: Venv unter `/tmp/kilo/docs-venv`, `mkdocs build --strict -d /tmp/kilo/docs-site`; danach `<pre class="mermaid">` in `architecture/index.html` prüfen (der Strict-Build beweist das Rendern nicht). Der `documenter` hat keine Shell, der Build ist ein Folge-Task für den `coder`.
- Der `reviewer` hat keine Shell: Verifikationsläufe wörtlich in den Auftrag schreiben.
- Der `tester` kann Produktivdateien nicht per `edit` ändern. Mutationsproben laufen über Kopien in `/tmp` und `cp`; Rücknahme per `sha256sum -c`. Nie `git checkout`/`stash`/`reset` auf einem uncommitteten Arbeitsbaum.
- Der planner darf keine zusammengesetzten Shell-Befehle (`&&`, `cd … &&`) ausführen und keinen `background_process`; einzelne `git`-Befehle und `gh pr create` gehen nach Rückfrage. `git fetch` und `git push` sind für Agenten gesperrt bzw. brauchen den Menschen (siehe `~/.config/kilo/RULES.md`).
- Subagent-Sessions können bei sehr langem Kontext mit API-Fehler 500 abbrechen; dann einen frischen Agenten auf dem vorhandenen Dateistand starten statt dieselbe Session zu wiederholen.
- Befehle für Subagenten immer einzeln, ohne `&&`.

## Aufgaben (aus dem Plan, T16–T22)

Siehe Tabelle „Phase 3 – Betrieb“ im Plan. Reihenfolge: T16 → T17/T18 → T19 → T20 (Review) → T21 (Doku) → T22 (Release `v1.0.0`, nur nach Bestätigung des Menschen).

Vorschlag für die Zerlegung beim Start:

- T16 in zwei Aufträge teilen: (a) `/health` + `healthcheck` (Go, mit Tests), (b) Dockerfile + `compose.yaml` (Image-Build und Lauf mit `read_only: true` prüfen, falls Docker verfügbar ist; sonst als offenen Punkt für den Menschen markieren).
- T17: `store.VacuumInto` mit Test (auch bei schreibgeschütztem Wurzelsystem simulierbar durch Schreiben in den Datenordner), Handler `POST /backup`, Download-Header, Aufräumen der temporären Datei (auch bei Fehlern und Abbruch), Test, dass die Datei eine gültige SQLite-Datei ist (`PRAGMA integrity_check`).
- T18: GitHub-Actions-Workflow (build, vet, test, optional mkdocs build --strict), Go-Version aus `go.mod`.
- Akzeptanzkriterien Phase 3: siehe Plan („`docker compose up` startet … `read_only: true`“, „Backup-Download liefert gültige SQLite-Datei“, „`TZ=Europe/Zurich` im distroless-Image“).

## Skill-Inbox

Nach Phase 2 liegen 10 Vorschläge in `.kilo/skill-proposals/` (nicht versioniert, lokal). Der `skiller` bearbeitet sie vor dem Start von Phase 3; Ergebnis (PR-URLs) steht im Plan-Log. Neue Vorschläge aus Phase 3 kommen nach Phase 3 an den skiller.

## Fehlende Rechte, die der Mensch entscheiden kann

- `tester`: `edit` auf Produktivcode nur für die Mutationsphase.
- `reviewer` und `documenter`: Shell (RULES.md: Sache des Menschen).
