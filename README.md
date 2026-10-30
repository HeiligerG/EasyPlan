# EasyPlan

EasyPlan ist eine self-hosted Mini-App in Go, die Fixkosten zusammenrechnet. Sie beantwortet drei Fragen:

1. Wie viel Geld brauche ich pro Monat, Woche und Tag?
2. Wie viel muss pro Konto in den nächsten N Tagen mindestens vorhanden sein?
3. Welche Zahlungen kommen als Nächstes?

Erfasst werden nur Ausgaben. Es gibt keine Einnahmen, keine Kontostände und keine Bank-Anbindung. Die Währung ist fest CHF.

## Status

**Phase 1 von 3. Die App ist noch nicht lauffähig.**

| Bereich | Stand |
|---|---|
| Berechnung (`internal/plan`) | fertig, getestet |
| Datenspeicher (`internal/store`) | fertig, getestet |
| Benutzeroberfläche (`internal/web`) | folgt in Phase 2 |
| Betrieb, Backup, Installation | folgt in Phase 3 |

`go run ./cmd/easyplan` liest heute nur die Konfiguration aus und beendet sich wieder. Es läuft noch kein Server.

## Leitplanken

- Ein Binary, eine SQLite-Datei, keine weiteren Dienste.
- Go mit Standardbibliothek, SQLite über `modernc.org/sqlite` ohne CGO, HTMX und CSS als eingebettete Dateien. Kein Node, kein Build-Schritt fürs Frontend, kein ORM.
- Konfiguration über Umgebungsvariablen, keine Einstellungsseite.
- Keine eigene Authentifizierung, der Zugriff regelt ein Reverse Proxy.
- Berechnungen entstehen bei Bedarf live, es werden keine abgeleiteten Daten gespeichert.

## Entwicklung

Go 1.27.3 oder neuer ist nötig.

```bash
# Prüfungen
go vet ./...
go test ./...

# Starten (Daten in ./data, Server folgt mit Phase 2)
export EASYPLAN_DATA=./data
go run ./cmd/easyplan
```

Die Umgebungsvariablen: `EASYPLAN_ADDR` (Standard `:8080`), `EASYPLAN_DATA` (Standard `/data`) und `TZ` (Standard `Europe/Zurich`).

## Dokumentation

Die Doku liegt unter [`docs/`](docs/) und wird mit [MkDocs](https://www.mkdocs.org/) und dem Material-Theme gebaut.

```bash
python3 -m venv .venv-docs
. .venv-docs/bin/activate
pip install -r requirements-docs.txt
mkdocs serve     # Vorschau auf http://127.0.0.1:8000
mkdocs build --strict
```

Inhalt:

- [Überblick und Status](docs/index.md)
- [Architektur](docs/architecture.md)
- [Berechnung](docs/features/berechnung.md)
- [Wiederholungen](docs/features/wiederholungen.md)
- [Datenspeicher](docs/features/datenspeicher.md)
- [Architekturentscheidungen](docs/adr/index.md)
- [Changelog](docs/changelog.md)

## Betrieb

Installation, Container, Datensicherung, Wiederherstellung und ein Beispiel für einen Reverse Proxy mit TLS folgen mit Phase 3. Es gibt dazu heute noch keine Anleitung, weil es noch nichts zu installieren gibt.

## Lizenz

MIT, siehe [LICENSE](LICENSE). Copyright (c) 2026 HeiligerG.
