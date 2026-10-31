# EasyPlan

EasyPlan ist eine self-hosted Mini-App in Go, die Fixkosten zusammenrechnet. Sie beantwortet drei Fragen:

1. Wie viel Geld brauche ich pro Monat, Woche und Tag?
2. Wie viel muss pro Konto in den nächsten N Tagen mindestens vorhanden sein?
3. Welche Zahlungen kommen als Nächstes?

Erfasst werden nur Ausgaben. Es gibt keine Einnahmen, keine Kontostände und keine Bank-Anbindung. Die Währung ist fest CHF.

## Status

**Phase 2 von 3. Die App ist benutzbar, der Betrieb folgt mit Phase 3.**

| Bereich | Stand |
|---|---|
| Berechnung (`internal/plan`) | fertig, getestet |
| Datenspeicher (`internal/store`) | fertig, getestet |
| Benutzeroberfläche (`internal/web`) | fertig, getestet |
| Server und Konfiguration (`cmd/easyplan`) | fertig, getestet |
| Betrieb, Backup, Installation | folgt in Phase 3 |

`go run ./cmd/easyplan` startet den Server und legt die Datenbank im Datenordner an. Es gibt noch keinen Healthcheck, kein Container-Image und keine Installationsanleitung, das folgt mit Phase 3.

## Leitplanken

- Ein Binary, eine SQLite-Datei, keine weiteren Dienste.
- Go mit Standardbibliothek, SQLite über `modernc.org/sqlite` ohne CGO, HTMX und CSS als eingebettete Dateien. Kein Node, kein Build-Schritt fürs Frontend, kein ORM.
- Konfiguration über Umgebungsvariablen, keine Einstellungsseite.
- Keine eigene Authentifizierung, der Zugriff regelt ein Reverse Proxy.
- Berechnungen entstehen bei Bedarf live, es werden keine abgeleiteten Daten gespeichert.

## Funktionen

Die Oberfläche besteht aus vier Seiten:

- **Dashboard** (`/`): Bedarf pro Monat, Woche und Tag, Reserve für die nächsten 7, 30, 60 oder 90 Tage und die nächste Zahlung, je Konto und als Total.
- **Einträge** (`/entries`): alle Einträge, nach Konto gruppiert. Anlegen, bearbeiten mit einem Klick auf die Zeile und löschen mit einer Bestätigungsseite.
- **Timeline** (`/timeline`): die Zahlungen der nächsten zwölf Monate, nach Monat gruppiert, mit der Summe je Monat und einem Filter auf das Konto.
- **Konten** (`/accounts`): anlegen, umbenennen, in der Reihenfolge verschieben, aktivieren und deaktivieren.

Alle Seiten funktionieren ohne JavaScript. Die Beträge stehen in ganzen Franken, die Daten im Format `TT.MM.JJJJ`.

## Entwicklung starten

Go 1.27.3 oder neuer ist nötig.

```bash
# Prüfungen
go vet ./...
go test ./...

# App starten, erreichbar unter http://127.0.0.1:8080
EASYPLAN_ADDR=127.0.0.1:8080 EASYPLAN_DATA=./data go run ./cmd/easyplan
```

Die Umgebungsvariablen:

| Variable | Standard | Bedeutung |
|---|---|---|
| `EASYPLAN_ADDR` | `:8080` | Listen-Adresse |
| `EASYPLAN_DATA` | `/data` | Ordner der SQLite-Datei `easyplan.db` |
| `TZ` | `Europe/Zurich` | Zeitzone |

Der Datenordner wird beim Start angelegt, wenn er noch nicht existiert. Eine ungültige Zeitzone bricht den Start mit einer Meldung ab.

> **Achtung:** Der Standardwert `EASYPLAN_ADDR=:8080` hört auf **allen** Schnittstellen des Rechners. EasyPlan hat keine eigene Authentifizierung, der Zugriff soll ein Reverse Proxy regeln. Für die lokale Entwicklung darum `127.0.0.1:8080` setzen, wie oben.

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
- [Oberfläche](docs/features/oberflaeche.md)
- [Dashboard](docs/features/dashboard.md)
- [Einträge](docs/features/eintraege.md)
- [Timeline](docs/features/timeline.md)
- [Konten](docs/features/konten.md)
- [Formate](docs/features/formate.md)
- [Server und Betrieb](docs/features/server.md)
- [Sicherheit](docs/features/sicherheit.md)
- [Berechnung](docs/features/berechnung.md)
- [Wiederholungen](docs/features/wiederholungen.md)
- [Datenspeicher](docs/features/datenspeicher.md)
- [Architekturentscheidungen](docs/adr/index.md)
- [Changelog](docs/changelog.md)

## Betrieb

Installation, Container, Datensicherung, Wiederherstellung und ein Beispiel für einen Reverse Proxy mit TLS folgen mit Phase 3. Es gibt dazu heute noch keine Anleitung, weil es noch nichts zu installieren gibt.

Wichtig für den Betrieb: Die Anwendung prüft bei jedem schreibenden Request, dass der `Origin`-Header zum `Host` passt. Ein Reverse Proxy muss darum den `Host`-Header erhalten, sonst sind alle Schreibzugriffe gesperrt. Die Einzelheiten stehen unter [Sicherheit](docs/features/sicherheit.md).

## Lizenz

MIT, siehe [LICENSE](LICENSE). Copyright (c) 2026 HeiligerG.
