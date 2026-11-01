# Übergabe: Phase 4 (optional) und Abschluss V1

Erstellt: 2026-11-01 · Gehört zu `.kilo/plans/2026-10-30-easyplan-v1.md` (Phasen 1–3 done, Tasks T22/T23 offen)

Diese Datei ist für eine neue planner-Session gedacht. Subagenten und die neue Session kennen den bisherigen Chat nicht. Lies zuerst diese Datei, dann den Plan (Tabellen „Tasks“ und „Entscheide“, Log ab 01.11.), dann `AGENTS.md` und `Konzept.md` (§10–§12).

## Startbedingung (macht der Mensch, der planner darf nicht fetchen)

1. `git checkout master`, `git pull` (lokaler `master` steht auf `e34fbea`, das Remote auf `8913376`, Merge von PR #7).
2. Den Docs-Abschluss-PR dieser Übergabe mergen (Branch `docs/phase-3-abschluss`: Plan und diese Datei), danach erneut `git pull`. Die Skill-Inbox `.kilo/skill-proposals/` ist per `.gitignore` nur lokal und liegt im selben Arbeitsverzeichnis.
3. Erst dann neuen Branch anlegen (Namen siehe „Vorgehen Phase 4“).

## Stand nach Phase 3

- `master` = `8913376`. Alle drei Phasen sind gemergt (PR #4/#5, #6, #7). Remote-Branches `feat/*` sind gelöscht.
- Qualität zuletzt (Phase-3-Stand): `gofmt` leer, `go vet` grün, `go test ./... -race` grün, 343 Top-Level-Tests, Coverage web 93.4 %, store 83.9 %, cmd 82.7 %; `mkdocs build --strict` Exit 0 (Python 3.11 und 3.12); CI auf GitHub grün (Jobs `Go (format, vet, build, test)`, `Vulnerability scan (govulncheck)`, `Docker image`, `Docs (MkDocs, strict)`); Container mit `read_only: true` verifiziert.
- Funktionen: Dashboard, Einträge, Konten, Timeline, `GET /health`, `easyplan healthcheck`, `POST /backup` (temporäre Kopie im Datenordner, Aufräumen auch beim Start), Dockerfile (distroless, Digest-gepinnt), `compose.yaml` (`127.0.0.1:8080:8080`, `read_only`), CI, Dependabot.
- Konfiguration: `EASYPLAN_ADDR` (ausserhalb Docker `127.0.0.1:8080`, im Image `:8080`), `EASYPLAN_DATA` (Default `/data`), `TZ` (Default `Europe/Zurich`).
- Noch nicht gemacht: Release `v1.0.0` (T22), Dependabot-Nachbesserung (T23), alles aus Konzept §11.

## Offene Punkte beim Menschen (nicht vom planner lösbar)

| Punkt | Hinweis |
|---|---|
| Dependabot-PRs #8, #9, #10 (pip) | #10 (`mkdocs-material` >=9.8.0) ist rot, weil `requirements-docs.txt` `pymdown-extensions<11` pinnt (belegt von D1: ResolutionImpossible). Reihenfolge: #8 mergen, dann bei #10 `@dependabot rebase`, danach #10 mergen; #9 unabhängig. Die Kombination aller drei baut grün. Alternativ T23 abwarten. |
| kilo-Repo (Skills) | PR #19 (`skill-authoring` 1.4 mit „Lieferung“, `agent-authoring` 1.1) mergen; danach `skiller-pr.sh sync`, `/reload` oder neue Session. Der Live-Checkout `~/.config/kilo` ist 34 Commits hinter `origin/main` und hat nicht committete Änderungen: vorher sichern oder verwerfen. Ohne Sync gelten die neuen Skills (`planner-urteil`, `agenten-fuehrung`, `ci-*`, `dependabot-konfiguration`) in der neuen Session NICHT automatisch. |
| `kilo.jsonc` | Fünf maschinenspezifische Pfade (`/home/developer/…`, Zeilen 554, 555, 576, 595, 614). Konfiguration, nur der Mensch. |
| Rechte des planner | Entwurf in `.kilo/skill-proposals/2026-11-01-planner-rechte-evaluation.md` (Option A empfohlen, Option B konservativ). Erst nach Umsetzung darf der planner z. B. `gh pr checks` und `git fetch/pull --ff-only`. Nie „Immer erlauben“ bei `git`/`gh` klicken. |
| Kernagent-Entwürfe | `tester-agentendefinition-ci` und `planner-agentendefinition-urteil` in `.kilo/skill-proposals/`: ohne diese Änderungen schreibt weiter der coder die CI und der planner lädt die neuen Skills nicht von sich aus. |
| Prüfskript im kilo-Repo | `scripts/validate_skills.py` prüft den Index nur auf Namen, nicht auf Versionen (Inbox: `verbesserungsvorschlag-validate-index-version`). |
| Historie | Commit `7470816` im kilo-Repo (PR #6) nennt den Projektnamen; Bereinigung nur per Force-Push, nicht empfohlen. |
| GitHub Pages | `site_url` in `mkdocs.yml` ist `https://heiligerg.github.io/EasyPlan/`, die Veröffentlichung ist nicht eingerichtet. |
| Gestaltung (E9) | Einträge-Tabelle bei 360 px nur seitlich scrollbar (672 px in 294 px Wrapper), Eingabefelder 39–44 px hoch (44-px-Regel gilt nur für Buttons), kein Favicon (404 im Proxy-Log). |
| Lokale Docker-Reste | Image `golang:1.27.3-bookworm` (~845 MB), `python:3.12-slim`, Build-Cache (~800 MB). `docker image prune`/`docker builder prune` entscheidet der Mensch. |

## Aufgaben vor und neben Phase 4

1. **T23** (coder, klein): in `.github/dependabot.yml` die pip-Gruppe `docs-tools` ohne `update-types`-Einschränkung (Docs-Tools sind gekoppelt, Entscheid E19; Hypothese zur Ursache der Einzel-PRs ist unbelegt, im Bericht ehrlich als solche kennzeichnen). Im Docs-Job von `ci.yml` vor dem Build `pip install --dry-run -r requirements-docs.txt`. `actionlint`, Schema-Validierung, Befehle lokal belegen. Danach `docs/ci.md` nachziehen (documenter).
2. **T22a/T22** (Release `v1.0.0`, erst nach T23 und Bestätigung des Menschen): Changelog `[Unreleased]` → `[1.0.0] – <Datum>`, neuer leerer `[Unreleased]` (documenter, Build durch coder). Tag und GitHub-Release macht der Mensch (`git tag`, `gh release create` sind für den planner gesperrt): Befehle in der Antwort nennen. Erst danach Phase 4.
3. **Dependabot-Nachprüfung** nach dem ersten Patch-/Minor-Update: greift die `golang`-Regel (Patch ja, Minor/Major nein) mit der gepinnten FROM-Zeile (`golang:1.27.3-bookworm@sha256:…`)? Werden die `# vX.Y.Z`-Kommentare hinter den Action-SHAs gepflegt? Beides ist noch unbelegt.

## Phase 4: Umfang (Konzept §11, nur bei Bedarf)

Alle Punkte sind klein und unabhängig. §10 hält sie ausdrücklich aus V1 heraus; sie kommen erst nach `v1.0.0`.

| Nr. | Punkt | Kern | Offene Entwurfsfragen (mit dem Menschen klären, nicht raten) |
|---|---|---|---|
| 1 | Benachrichtigung | Umgebungsvariable `EASYPLAN_NOTIFY_URL` (https, z. B. ntfy/Gotify); täglicher Check sendet „Zahlung in ≤ 3 Tagen“, sonntags Wochenzusammenfassung; kein Retry, kein Log, keine UI; Fehler ins Server-Log; keine Kontostände oder Notizen in der Nachricht | Dürfen Beträge und Eintragsnamen in die Nachricht? (Konzept sagt: keine Kontostände oder Notizen.) Uhrzeit des täglichen Checks, Zeitzone (`TZ`), Verhalten beim Neustart (doppelte Nachricht vermeiden ohne gespeicherten Zustand?), Ziel-URL nur `https`, Timeouts, SSRF-Aspekte (die URL setzt der Betreiber), Testbarkeit (Uhr injizierbar, `internal/plan` bleibt rein) |
| 2 | CSV-Export der Einträge | Download aller (aktiven?) Einträge als CSV | Methode (`GET` darf nach Entscheid E7 nur sicher sein: ein Export ist lesend, liefert aber alle Daten: Begründung festhalten), Spalten und Reihenfolge, Beträge in CHF mit zwei Dezimalen oder Rappen, Datumsformat, Trennzeichen und Zeichensatz (Excel-Kompatibilität), CSV-Injection (Zellen mit `=`, `+`, `-`, `@` am Anfang) |
| 3 | Dark Mode | nur CSS per `prefers-color-scheme` | Kontrastwerte (WCAG AA wie bei den Hover-Farben), Fokus-Ringe, Tabellen/Formulare, CSP bleibt `default-src 'self'`; Messung bei 360 px |
| 4 | PWA-Manifest | installierbar, ohne Daten-Cache | Icons (Assets einbetten), Pflichtfelder des Manifests, ob für „installierbar“ ein Service Worker nötig ist (zum Zeitpunkt der Umsetzung belegen, nicht aus der Erinnerung), CSP-Auswirkung, Scope/`start_url` hinter einem Reverse Proxy mit Pfad-Präfix |

Vorschlag zur Reihenfolge: Konzept §11 sagt „nach Nutzen“; der planner fragt den Menschen beim Start, welche Punkte er will und in welcher Reihenfolge, und empfiehlt Dark Mode und CSV-Export als kleine Einstiegspunkte. Gestaltungsfragen aus E9 (Tabelle bei 360 px, Eingabefelder, Favicon) lassen sich mit Dark Mode zusammen in einem UI-Paket erledigen, wenn der Mensch das möchte.

## Vorgehen Phase 4

- Ein PR pro Punkt, eigener Branch, z. B. `feat/phase-4-dark-mode`, `feat/phase-4-csv-export`, `feat/phase-4-notify`, `feat/phase-4-pwa` (Grund: Punkte sind unabhängig, Konflikte vermeiden, Review klein halten). `AGENTS.md` nennt „ein Feature-Branch pro Phase“: Abweichung dem Menschen mitteilen; Änderungen an `AGENTS.md` macht der Mensch (Edit-Sperre der Agenten).
- Pro Punkt: coder → tester (inkl. Mutationsprobe, Messung bei 360 px für UI) → reviewer → documenter → coder (Doku-Build) → Commit/Push/PR durch den planner nach Freigabe. Plan-Datei: `.kilo/plans/<datum>-phase-4-<slug>.md` pro Punkt oder eine gemeinsame Datei mit Tabelle; vor dem Start mit dem Menschen bestätigen (mehr als zwei Tasks).
- Konzept.md nach Abschluss angleichen (verbindlich bleiben Code und `docs/`); der coder macht das, der documenter hat keinen Schreibzugriff darauf.
- Keine neuen Abhängigkeiten ohne Begründung (`AGENTS.md`); Standardbibliothek zuerst (Benachrichtigung: `net/http`).

## Arbeitsweise (gilt, auch wenn die neuen Skills noch nicht synchronisiert sind)

Der Mensch hat im Verlauf von Phase 3 festgelegt:

- **Selbst eingreifen und melden.** Ist eine Entscheidung umkehrbar, im Umfang, ohne Kosten, ohne neue Rechte und ohne Widerspruch zum Menschen, entscheidet der planner selbst und meldet sie im ersten Absatz der Zusammenfassung; sie kommt in die Tabelle „Entscheide“ (mit Grund, zweitbester Lösung, Umkehr). Fragen an den Menschen kommen mit Empfehlung. Offene Punkte aus Berichten enden mit einem Entscheid, nie mit „zur Kenntnis“. Fragen vor der Ausführung nur bei: neuem Umfang, Unumkehrbarem (Push, Merge, Release, Löschen), Kosten, neuen Rechten, Kernagenten und `RULES.md`.
- **Gnädig führen.** Nachbesserungsaufträge in der Reihenfolge Anerkannt, Änderung, Warum, Prüfung. Abbrüche sind nie Schuld des Agenten; Lücke im Auftrag, in den Rechten oder im Werkzeug suchen. Gemeldete Abweichungen prüfen und würdigen; nicht gemeldete Rechteumgehungen sachlich benennen, Ursache im System suchen, nie als Regel übernehmen.
- **Aufträge mit SPIELRAUM** (selbst entscheiden / melden und entscheiden / zurückfragen) und Rückgabe mit EMPFEHLUNGEN vor OFFENE PUNKTE; jede Empfehlung wird im Log beantwortet (angenommen, abgelehnt mit Grund, wartet auf Mensch).
- **Vor jeder Delegation die Rechte der Rolle prüfen.** `git commit`/`git push` darf nur der planner (nach Freigabe des Menschen); coder, tester, documenter, reviewer committen nicht. Fehlt einer Rolle ein Recht, `STATUS: blocked` mit Rechteliste oder anderer Rolle beauftragen, keine Umgehung.
- **Eine Lieferung an das kilo-Repo ist ein PR.** Gemeinsame Dateien nur gestapelt, Index nicht parallel ändern, Projektbezug-Check vor dem ersten Öffnen; keine Projektnamen oder Task-IDs in globalen Skills und deren PR-Texten.
- **CI und Dependabot** gehören zum Testen und entstehen mit den ersten Tests eines neuen Artefakts (neuer Workflow-Job, neues Ökosystem). Bis die Kernagenten angepasst sind, schreibt der coder sie nach Vorgabe des planners. „CI grün“ prüft der planner vor „bereit zum Merge“ über die GitHub-MCP-Werkzeuge (`get_check_runs`); `gh pr checks` ist gesperrt.

## Arbeitsumgebung (Erfahrungen)

- Go liegt unter `~/sdk/go` (1.27.3); wenn `go` nicht im PATH ist, Pfad ergänzen. `actionlint` unter `~/go/bin`.
- Docs-Build: Venv unter `/tmp/kilo/docs-venv` (kann fehlen), `mkdocs build --strict -d /tmp/kilo/docs-site`; Mermaid prüfen (`<pre class="mermaid">` in `architecture/index.html`), Admonitions und Tabellen im HTML; zusätzlich unter Python 3.12 per `docker run --rm -v <Repo>:/src:ro python:3.12-slim …` (wie die CI). Der documenter hat keine Shell: Build und Link-Prüfung nach Dateiebene (`../` nur in `docs/features/` und `docs/adr/`) als Folge-Task an den coder.
- Reviewer hat keine Shell: Verifikationsläufe wörtlich in den Auftrag.
- Tester kann Produktivdateien nicht per `edit` ändern: Mutationsproben in Kopien unter `/tmp/kilo/…` (Python + `cp`), Nachweis per `sha256sum -c`; Compile-Fehler gelten nicht als erkannte Mutation. Nie `git checkout`/`stash`/`reset` auf einem uncommitteten Arbeitsbaum.
- Parallele Docker-Aufträge: eigener Image-Tag (z. B. `easyplan:t<nr>-<rolle>`), nur eigene Images entfernen, kein `prune`.
- planner-Shell: einzelne Befehle ohne `&&`, Verzeichnis über `workdir`, kein `cd`; erlaubt waren zuletzt `git status/diff/log/branch/add/commit/checkout -b`, `git push -u origin <branch>`, `gh pr list/view/create`; gesperrt `git fetch/pull`, `gh pr checks`, `which`, `docker`, `background_process`. Branch-Namen `feat/…` und `docs/…` funktionieren für Push nach Rückfrage.
- Subagent-Sessions können bei langem Kontext mit API-Fehler abbrechen: erst Arbeitsstand prüfen (`git status --short`), dann einmal mit `task_id` fortsetzen, sonst frischen Agenten mit knappem Nachfolger-Auftrag starten. Das `skill`-Tool nimmt genau einen Parameter `name`.
- Der skiller hat keine Shell und kein Merge-Recht; sein GitHub-Zugriff auf das kilo-Repo liefert 404. Der planner sieht das kilo-Repo ebenfalls nicht (404 über die MCP-Werkzeuge): Kontrolle dort macht der Mensch oder ein coder mit `gh`.

## Skill-Inbox

- `.kilo/skill-proposals/` enthält aktuell nur noch die zwei Kernagent-Entwürfe, die Rechte-Evaluation und den Verbesserungsvorschlag zum Prüfskript (alles beim Menschen); übrige Vorschläge sind in `processed/` (mit Entscheidungszeile) und als kilo-PRs #7–#19 gemergt bzw. (#19) zum Merge bereit.
- Neue Vorschläge aus Phase 4 legt der planner/tester/coder in die Inbox; Weitergabe an den skiller pro Lieferung als ein Auftrag (er liefert einen PR).
