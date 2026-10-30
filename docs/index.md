# EasyPlan

EasyPlan ist eine self-hosted Mini-App in Go, die Fixkosten zusammenrechnet. Sie beantwortet drei Fragen:

1. Wie viel Geld brauche ich pro Monat, Woche und Tag?
2. Wie viel muss pro Konto in den nächsten N Tagen mindestens vorhanden sein?
3. Welche Zahlungen kommen als Nächstes?

Erfasst werden nur Ausgaben. Es gibt keine Einnahmen, keine Kontostände und keine Bank-Anbindung. Die Währung ist fest CHF.

## Status

Phase 1 von 3 ist abgeschlossen: die Berechnung und der Datenspeicher stehen und sind getestet. Die Benutzeroberfläche folgt mit Phase 2, der Betrieb mit Phase 3.

| Bereich | Paket | Stand |
|---|---|---|
| Berechnung | `internal/plan` | fertig, getestet |
| Datenspeicher | `internal/store` | fertig, getestet |
| Benutzeroberfläche | `internal/web` | leer, folgt in Phase 2 |
| Server und Konfiguration | `cmd/easyplan` | Gerüst, folgt in Phase 2 |
| Betrieb, Backup, Installation | – | folgt in Phase 3 |

!!! warning "Noch nicht als App nutzbar"
    Der Webserver läuft noch nicht. `go run ./cmd/easyplan` liest nur die Konfiguration aus und beendet sich wieder. Die erste lauffähige Fassung gibt es mit Phase 2.

## Leitplanken

- Ein Binary, eine SQLite-Datei, keine weiteren Dienste.
- Go mit Standardbibliothek, SQLite über `modernc.org/sqlite` ohne CGO, HTMX und CSS als eingebettete Dateien. Kein Node, kein Build-Schritt fürs Frontend, kein ORM.
- Konfiguration über Umgebungsvariablen, keine Einstellungsseite.
- Keine eigene Authentifizierung, der Zugriff regelt ein Reverse Proxy.
- Berechnungen entstehen bei Bedarf live, es werden keine abgeleiteten Daten gespeichert.

## Schnellstart

Go 1.27.3 oder neuer ist nötig (`go.mod`).

```bash
export EASYPLAN_DATA=./data
go run ./cmd/easyplan
```

Ausgabe:

```text
easyplan scaffold: listen=:8080 data=./data tz=Europe/Zurich
```

Der Befehl beendet sich danach. Er öffnet noch keine Datenbank und startet keinen Server, beides folgt mit Phase 2.

Die Umgebungsvariablen mit ihren Standardwerten:

| Variable | Standard | Bedeutung |
|---|---|---|
| `EASYPLAN_ADDR` | `:8080` | Listen-Adresse |
| `EASYPLAN_DATA` | `/data` | Ordner der SQLite-Datei |
| `TZ` | `Europe/Zurich` | Zeitzone |

## Weiterführend

- [Architektur](architecture.md): Bausteine, Datenfluss und Datumskonvention
- [Berechnung](features/berechnung.md): Monat, Woche, Tag, Reserve und Rundung
- [Wiederholungen](features/wiederholungen.md): Einheiten, Intervalle und Monatsende
- [Datenspeicher](features/datenspeicher.md): Schema, Migrationen und Fehler
- [Architekturentscheidungen](adr/index.md): die ADRs mit Kontext und Folgen
- [Beitragen](contributing.md): Branch, Prüfungen und Sprachregeln
- [Changelog](changelog.md): was sich geändert hat
