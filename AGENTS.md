# AGENTS.md

## Projektzweck

EasyPlan ist eine self-hosted Mini-App in Go, die Fixkosten pro Konto zusammenrechnet (Monat, Woche, Tag), die Reserve für die nächsten N Tage ausweist und kommende Zahlungen zeigt.
Es werden nur Ausgaben erfasst, keine Einnahmen und keine Kontostände.

Konzept.md ist die Quelle der Wahrheit für Datenmodell, Berechnung, Sicherheit und Betrieb.

## Leitplanken (Konzept.md §2)

- Ein Binary, eine SQLite-Datei, keine weiteren Dienste.
- Go mit Standardbibliothek, SQLite über `modernc.org/sqlite` (ohne CGO), HTMX und CSS als eingebettete Dateien. Kein Node, kein Frontend-Build, kein ORM.
- Konfiguration über Umgebungsvariablen, keine Einstellungsseite.
- Keine eigene Authentifizierung, Zugriff regelt ein Reverse Proxy.
- Berechnungen werden live gemacht, es werden keine abgeleiteten Daten gespeichert.

## Befehle

```sh
go vet ./...
go test ./...
EASYPLAN_DATA=./data go run ./cmd/easyplan
```

Später zusätzlich `mkdocs build --strict` (Doku liegt unter `docs/`).

## Code-Konventionen

- Standardbibliothek zuerst; neue Abhängigkeiten nur mit Begründung.
- Kein ORM, SQL direkt im Store-Paket.
- Beträge sind `int64` in Rappen, das Vorzeichen entsteht erst bei der Anzeige.
- `internal/plan` ist rein: kein I/O, kein `time.Now()`, "heute" wird übergeben.
- Datumswerte sind lokale Kalendertage (`YYYY-MM-DD`) ohne Uhrzeit.

## Branch- und PR-Regeln

- Ein Feature-Branch pro Phase (`feat/phase-1-kern`, `feat/phase-2-ui`, `feat/phase-3-betrieb`).
- Kein Push auf `master`, kein lokaler Merge: Zusammenführen läuft über einen GitHub-PR.
- Commit, Push und PR macht der planner nach Review und Bestätigung durch den Menschen.

## Sprache

- Dokumentation (README, MkDocs, Pläne): Deutsch mit Schweizer Schreibweise, ohne ß.
- Code, Bezeichner, Commit-Messages und Code-Kommentare: Englisch.
